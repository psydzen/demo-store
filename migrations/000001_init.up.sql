-- Users: admins and players share one table; the token is the only credential.
CREATE TABLE users (
    id         BIGSERIAL   PRIMARY KEY,
    name       TEXT        NOT NULL,
    token      TEXT        NOT NULL UNIQUE,
    role       TEXT        NOT NULL CHECK (role IN ('admin', 'player')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE quizzes (
    id          BIGSERIAL   PRIMARY KEY,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'draft'
                            CHECK (status IN ('draft', 'published', 'closed')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rounds (
    id       BIGSERIAL PRIMARY KEY,
    quiz_id  BIGINT    NOT NULL REFERENCES quizzes (id) ON DELETE CASCADE,
    title    TEXT      NOT NULL,
    position INT       NOT NULL
);

CREATE INDEX rounds_quiz_position_idx ON rounds (quiz_id, position);

CREATE TABLE questions (
    id             BIGSERIAL PRIMARY KEY,
    round_id       BIGINT    NOT NULL REFERENCES rounds (id) ON DELETE CASCADE,
    position       INT       NOT NULL,
    kind           TEXT      NOT NULL CHECK (kind IN ('choice', 'free')),
    text           TEXT      NOT NULL,
    points_correct INT       NOT NULL DEFAULT 1,
    points_wrong   INT       NOT NULL DEFAULT 0,
    admin_hint     TEXT      NOT NULL DEFAULT ''
);

CREATE INDEX questions_round_position_idx ON questions (round_id, position);

CREATE TABLE answer_options (
    id          BIGSERIAL PRIMARY KEY,
    question_id BIGINT    NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    text        TEXT      NOT NULL,
    is_correct  BOOLEAN   NOT NULL DEFAULT false,
    position    INT       NOT NULL
);

CREATE INDEX answer_options_question_position_idx
    ON answer_options (question_id, position);

CREATE TABLE attempts (
    id          BIGSERIAL   PRIMARY KEY,
    quiz_id     BIGINT      NOT NULL REFERENCES quizzes (id) ON DELETE CASCADE,
    user_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    UNIQUE (quiz_id, user_id)
);

-- is_correct and points_awarded stay NULL for free-text answers until an admin
-- reviews them; NULL is what puts a response into the review queue.
CREATE TABLE responses (
    id             BIGSERIAL   PRIMARY KEY,
    attempt_id     BIGINT      NOT NULL REFERENCES attempts (id) ON DELETE CASCADE,
    question_id    BIGINT      NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    option_id      BIGINT      REFERENCES answer_options (id) ON DELETE SET NULL,
    free_text      TEXT        NOT NULL DEFAULT '',
    is_correct     BOOLEAN,
    points_awarded INT,
    answered_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at    TIMESTAMPTZ,
    reviewed_by    BIGINT      REFERENCES users (id) ON DELETE SET NULL,
    UNIQUE (attempt_id, question_id)
);

CREATE INDEX responses_pending_idx ON responses (id) WHERE is_correct IS NULL;

-- Session store for alexedwards/scs.
CREATE TABLE sessions (
    token  TEXT        PRIMARY KEY,
    data   BYTEA       NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

-- Bootstrap admin. Override the token with the ADMIN_TOKEN env var.
INSERT INTO users (name, token, role)
VALUES ('Admin', 'change-me-admin-token-01', 'admin');
