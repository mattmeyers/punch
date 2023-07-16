CREATE TABLE `profile` (
    `id` INTEGER PRIMARY KEY AUTOINCREMENT,
    `slug` TEXT NOT NULL UNIQUE,
    `description` TEXT NULL DEFAULT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO `profile` (`id`, `slug`) VALUES (1, 'default');

CREATE TABLE `session` (
    `start` DATETIME NOT NULL,
    `stop` DATETIME NULL DEFAULT NULL,
    `note` TEXT NULL DEFAULT NULL,
    `profile_id` INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (`profile_id`) REFERENCES `profile` (`id`)
);

