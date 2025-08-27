-- Каскадное удаление теории и факта при удалении поста
ALTER TABLE channel_post_theory
    DROP CONSTRAINT IF EXISTS channel_post_theory_channel_post_id_fkey;
ALTER TABLE channel_post_theory
    ADD CONSTRAINT channel_post_theory_channel_post_id_fkey
        FOREIGN KEY (channel_post_id) REFERENCES channel_post(id) ON DELETE CASCADE;

ALTER TABLE channel_post_fact
    DROP CONSTRAINT IF EXISTS channel_post_fact_channel_post_theory_id_fkey;
ALTER TABLE channel_post_fact
    ADD CONSTRAINT channel_post_fact_channel_post_theory_id_fkey
        FOREIGN KEY (channel_post_theory_id) REFERENCES channel_post_theory(id) ON DELETE CASCADE;
