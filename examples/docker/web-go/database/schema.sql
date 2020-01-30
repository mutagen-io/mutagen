-- Create the messages table if it doesn't already exist.
CREATE TABLE IF NOT EXISTS messages (
    -- id is the message id.
    id BIGSERIAL PRIMARY KEY,
    -- submitted_at is the submission time for the message.
    submitted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- name is the message submitter's name.
    name VARCHAR(50) NOT NULL,
    -- message is the message itself.
    message VARCHAR(200) NOT NULL
);
