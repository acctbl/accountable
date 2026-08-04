-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.accountable_schema_state (
    singleton boolean PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    version bigint NOT NULL,
    dirty boolean NOT NULL
);

INSERT INTO public.accountable_schema_state (singleton, version, dirty)
VALUES (TRUE, 0, FALSE)
ON CONFLICT (singleton) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE public.accountable_schema_state;
-- +goose StatementEnd
