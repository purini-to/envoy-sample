-- 2019-06-19 13:50:27

CREATE TABLE `task` (
  `id` VARCHAR(20) NOT NULL,
  `title` VARCHAR(50) NOT NULL,
  `state` VARCHAR(20) NOT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT current_timestamp,
  `updated_at` TIMESTAMP NOT NULL DEFAULT current_timestamp on update current_timestamp,
  PRIMARY KEY (`id`));

INSERT INTO `task` (`id`, `title`, `state`) VALUES ('bk4v5l23q56drr9kh9ig', 'conference', 'completed');
INSERT INTO `task` (`id`, `title`, `state`) VALUES ('bk4v6023q56ds23c1jl0', 'recap', 'uncompleted');
