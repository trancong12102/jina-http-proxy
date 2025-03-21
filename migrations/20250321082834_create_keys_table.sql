-- +goose Up
-- +goose StatementBegin
CREATE TABLE "keys" (
	"key" varchar PRIMARY KEY NOT NULL,
	"balance" integer NOT NULL,
	"used_at" timestamp with time zone,
	"created_at" timestamp with time zone NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "keys";
-- +goose StatementEnd
