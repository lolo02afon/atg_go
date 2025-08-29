-- Переименование полей источника в channel_duplicate
ALTER TABLE channel_duplicate
    RENAME COLUMN url_channel_duplicate TO url_channel_donor;
ALTER TABLE channel_duplicate
    RENAME COLUMN channel_duplicate_tgid TO channel_donor_tgid;

