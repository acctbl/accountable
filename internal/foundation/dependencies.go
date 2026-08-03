package foundation

import (
	"context"
	"errors"
)

var ErrFoundationUnavailable = errors.New("foundation dependencies are unavailable")

type Dependencies struct {
	Database *Database
	Storage  Storage
	Crypto   Crypto
}

func Build(ctx context.Context, config Config) (*Dependencies, error) {
	resolver, err := NewFileSecretResolver(config.Secrets.Directory)
	if err != nil {
		return nil, ErrFoundationUnavailable
	}
	database, err := OpenDatabase(ctx, config.Database, resolver)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Dependencies, error) {
		database.Close()
		return nil, err
	}
	storage, err := NewFileStorage(config.Storage.Root)
	if err != nil {
		return fail(err)
	}
	key, err := resolver.Resolve(ctx, config.Crypto.KeyRef)
	if err != nil {
		return fail(ErrCryptoUnavailable)
	}
	cryptor, err := NewLocalCrypto(key)
	if err != nil {
		return fail(err)
	}
	dependencies := &Dependencies{Database: database, Storage: storage, Crypto: cryptor}
	checkCtx, cancel := context.WithTimeout(ctx, config.Database.ConnectTimeout)
	defer cancel()
	if err := dependencies.Check(checkCtx); err != nil {
		return fail(err)
	}
	return dependencies, nil
}

func (d *Dependencies) Check(ctx context.Context) error {
	if err := d.Database.Check(ctx); err != nil {
		return err
	}
	if err := d.Storage.Check(ctx); err != nil {
		return err
	}
	if err := d.Crypto.Check(ctx); err != nil {
		return err
	}
	return nil
}

func (d *Dependencies) Close() { d.Database.Close() }
