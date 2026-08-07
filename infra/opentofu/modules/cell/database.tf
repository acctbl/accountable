resource "aws_db_subnet_group" "cell" {
  name       = local.name
  subnet_ids = [for subnet in aws_subnet.database : subnet.id]
}

resource "aws_db_instance" "cell" {
  allocated_storage               = 20
  apply_immediately               = true
  auto_minor_version_upgrade      = true
  backup_retention_period         = var.cell_lifecycle == "durable" ? 35 : 1
  ca_cert_identifier              = "rds-ca-rsa2048-g1"
  copy_tags_to_snapshot           = true
  db_name                         = "accountable"
  db_subnet_group_name            = aws_db_subnet_group.cell.name
  delete_automated_backups        = var.cell_lifecycle != "durable"
  deletion_protection             = var.cell_lifecycle == "durable"
  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  engine                          = "postgres"
  engine_version                  = "18.4"
  identifier                      = local.name
  instance_class                  = "db.t4g.micro"
  kms_key_id                      = aws_kms_key.cell.arn
  manage_master_user_password     = true
  master_user_secret_kms_key_id   = aws_kms_key.cell.arn
  multi_az                        = true
  port                            = local.database_port
  publicly_accessible             = false
  final_snapshot_identifier       = var.cell_lifecycle == "durable" ? var.database_final_snapshot_identifier : null
  skip_final_snapshot             = var.cell_lifecycle != "durable"
  storage_encrypted               = true
  storage_type                    = "gp3"
  username                        = "accountable_admin"
  vpc_security_group_ids          = [aws_security_group.database.id]

  lifecycle {
    precondition {
      condition     = var.environment == "development" || var.cell_lifecycle == "durable"
      error_message = "Staging and production cells must use the durable lifecycle."
    }

    precondition {
      condition = (
        var.cell_lifecycle != "durable" ||
        try(trimspace(var.database_final_snapshot_identifier), "") != ""
      )
      error_message = "Durable databases require a unique final snapshot identifier."
    }

    postcondition {
      condition = (
        self.multi_az &&
        !self.publicly_accessible &&
        self.storage_encrypted &&
        (
          var.cell_lifecycle != "durable" ||
          (
            self.backup_retention_period == 35 &&
            self.deletion_protection &&
            !self.delete_automated_backups &&
            !self.skip_final_snapshot
          )
        )
      )
      error_message = "RDS must remain private, encrypted, Multi-AZ, and durable environments must retain 35-day PITR plus deletion safeguards."
    }
  }
}
