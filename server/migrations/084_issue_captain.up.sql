-- Add per-issue routing agent ("captain"). Polymorphic pair mirrors
-- project.lead_type/lead_id; v1 only allows captain_type='agent', v2 will
-- relax the CHECK to add 'playbook'. No FK on captain_id (matches
-- project.lead_id) — existence is validated at the handler layer.
ALTER TABLE issue
    ADD COLUMN captain_type TEXT CHECK (captain_type IN ('agent')),
    ADD COLUMN captain_id UUID;

ALTER TABLE issue
    ADD CONSTRAINT issue_captain_pair_check
    CHECK ((captain_type IS NULL AND captain_id IS NULL)
        OR (captain_type IS NOT NULL AND captain_id IS NOT NULL));

CREATE INDEX idx_issue_captain ON issue(captain_type, captain_id)
    WHERE captain_id IS NOT NULL;
