CREATE TABLE payments (
    id         bigserial PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    quiz_id    bigint      NOT NULL REFERENCES quizzes (id) ON DELETE CASCADE,
    amount     bigint      NOT NULL,
    currency   text        NOT NULL,
    card_last4 text        NOT NULL,
    status     text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX payments_user_id_idx ON payments (user_id);
