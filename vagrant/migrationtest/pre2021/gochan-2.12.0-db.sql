--
-- gochan 2.12.0 sample MySQL database for migration testing
-- TODO: Make it compatible with Postgres (not a priority since it's primarily for testing migration)
--

CREATE DATABASE gochan_pre2021_db;
USE gochan_pre2021_db;

CREATE TABLE `gc_announcements` (
	`id` SERIAL,
	`subject` VARCHAR(45) NOT NULL DEFAULT '',
	`message` TEXT NOT NULL CHECK (message <> ''),
	`poster` VARCHAR(45) NOT NULL CHECK (poster <> ''),
	`timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_appeals` (
	`id` SERIAL,
	`ban` INT(11) UNSIGNED NOT NULL CHECK (ban <> 0),
	`message` TEXT NOT NULL CHECK (message <> ''),
	`timestamp` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	`denied` BOOLEAN DEFAULT false,
	`staff_response` TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_banlist` (
	`id` SERIAL,
	`allow_read` BOOLEAN DEFAULT TRUE,
	`ip` VARCHAR(45) NOT NULL DEFAULT '',
	`name` VARCHAR(255) NOT NULL DEFAULT '',
	`name_is_regex` BOOLEAN DEFAULT FALSE,
	`filename` VARCHAR(255) NOT NULL DEFAULT '',
	`file_checksum` VARCHAR(255) NOT NULL DEFAULT '',
	`boards` VARCHAR(255) NOT NULL DEFAULT '*',
	`staff` VARCHAR(50) NOT NULL DEFAULT '',
	`timestamp` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	`expires` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	`permaban` BOOLEAN NOT NULL DEFAULT TRUE,
	`reason` VARCHAR(255) NOT NULL DEFAULT '',
	`type` SMALLINT NOT NULL DEFAULT 3,
	`staff_note` VARCHAR(255) NOT NULL DEFAULT '',
	`appeal_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	`can_appeal` BOOLEAN NOT NULL DEFAULT true,
	PRIMARY KEY (id)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_boards` (
	`id` SERIAL,
	`list_order` TINYINT UNSIGNED NOT NULL DEFAULT 0,
	`dir` VARCHAR(45) NOT NULL CHECK (dir <> ''),
	`type` TINYINT UNSIGNED NOT NULL DEFAULT 0,
	`upload_type` TINYINT UNSIGNED NOT NULL DEFAULT 0,
	`title` VARCHAR(45) NOT NULL CHECK (title <> ''),
	`subtitle` VARCHAR(64) NOT NULL DEFAULT '',
	`description` VARCHAR(64) NOT NULL DEFAULT '',
	`section` INT NOT NULL DEFAULT 1,
	`max_file_size` INT UNSIGNED NOT NULL DEFAULT 4718592,
	`max_pages` TINYINT UNSIGNED NOT NULL DEFAULT 11,
	`default_style` VARCHAR(45) NOT NULL DEFAULT '',
	`locked` BOOLEAN NOT NULL DEFAULT FALSE,
	`created_on` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	`anonymous` VARCHAR(45) NOT NULL DEFAULT 'Anonymous',
	`forced_anon` BOOLEAN NOT NULL DEFAULT FALSE,
	`max_age` INT(20) UNSIGNED NOT NULL DEFAULT 0,
	`autosage_after` INT(5) UNSIGNED NOT NULL DEFAULT 200,
	`no_images_after` INT(5) UNSIGNED NOT NULL DEFAULT 0,
	`max_message_length` INT(10) UNSIGNED NOT NULL DEFAULT 8192,
	`embeds_allowed` BOOLEAN NOT NULL DEFAULT TRUE,
	`redirect_to_thread` BOOLEAN NOT NULL DEFAULT TRUE,
	`require_file` BOOLEAN NOT NULL DEFAULT FALSE,
	`enable_catalog` BOOLEAN NOT NULL DEFAULT TRUE,
	PRIMARY KEY (`id`),
	UNIQUE (`dir`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_embeds` (
	`id` SERIAL,
	`filetype` VARCHAR(3) NOT NULL,
	`name` VARCHAR(45) NOT NULL,
	`video_url` VARCHAR(255) NOT NULL,
	`width` SMALLINT UNSIGNED NOT NULL,
	`height` SMALLINT UNSIGNED NOT NULL,
	`embed_code` TEXT NOT NULL,
	PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_info` (
	`name` VARCHAR(45) NOT NULL,
	`value` TEXT NOT NULL,
	PRIMARY KEY (`name`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_links` (
	`id` SERIAL,
	`title` VARCHAR(45) NOT NULL,
	`url` VARCHAR(255) NOT NULL,
	PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_posts` (
	`id` SERIAL,
	`boardid` INT NOT NULL,
	`parentid` INT(10) UNSIGNED NOT NULL DEFAULT '0',
	`name` VARCHAR(50) NOT NULL,
	`tripcode` VARCHAR(10) NOT NULL,
	`email` VARCHAR(50) NOT NULL,
	`subject` VARCHAR(100) NOT NULL,
	`message` TEXT NOT NULL,
	`message_raw` TEXT NOT NULL,
	`password` VARCHAR(45) NOT NULL,
	`filename` VARCHAR(45) NOT NULL DEFAULT '',
	`filename_original` VARCHAR(255) NOT NULL DEFAULT '',
	`file_checksum` VARCHAR(45) NOT NULL DEFAULT '',
	`filesize` INT(20) UNSIGNED NOT NULL DEFAULT 0,
	`image_w` SMALLINT(5) UNSIGNED NOT NULL DEFAULT 0,
	`image_h` SMALLINT(5) UNSIGNED NOT NULL DEFAULT 0,
	`thumb_w` SMALLINT(5) UNSIGNED NOT NULL DEFAULT 0,
	`thumb_h` SMALLINT(5) UNSIGNED NOT NULL DEFAULT 0,
	`ip` VARCHAR(45) NOT NULL DEFAULT '',
	`tag` VARCHAR(5) NOT NULL DEFAULT '',
	`timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	`autosage` BOOLEAN NOT NULL DEFAULT FALSE,
	`deleted_timestamp` TIMESTAMP,
	`bumped` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	`stickied` BOOLEAN NOT NULL DEFAULT FALSE,
	`locked` BOOLEAN NOT NULL DEFAULT FALSE,
	`reviewed` BOOLEAN NOT NULL DEFAULT FALSE,
	PRIMARY KEY  (`boardid`,`id`),
	KEY `parentid` (`parentid`),
	KEY `bumped` (`bumped`),
	KEY `file_checksum` (`file_checksum`),
	KEY `stickied` (`stickied`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_reports` (
	`id` SERIAL,
	`board` VARCHAR(45) NOT NULL,
	`postid` INT(10) UNSIGNED NOT NULL,
	`timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	`ip` VARCHAR(45) NOT NULL,
	`reason` VARCHAR(255) NOT NULL,
	`cleared` BOOLEAN NOT NULL DEFAULT FALSE,
	`istemp` BOOLEAN NOT NULL DEFAULT FALSE,
	PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_sections` (
	`id` SERIAL,
	`list_order` INT UNSIGNED NOT NULL DEFAULT 0,
	`hidden` BOOLEAN NOT NULL DEFAULT FALSE,
	`name` VARCHAR(45) NOT NULL,
	`abbreviation` VARCHAR(10) NOT NULL,
	PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_sessions` (
	`id` SERIAL,
	`name` CHAR(16) NOT NULL,
	`sessiondata` VARCHAR(45) NOT NULL,
	`expires` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (`id`)
) ENGINE=MEMORY DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_staff` (
	`id` SERIAL,
	`username` VARCHAR(45) NOT NULL,
	`password_checksum` VARCHAR(120) NOT NULL,
	`rank` TINYINT(1) UNSIGNED NOT NULL DEFAULT 2,
	`boards` VARCHAR(128) NOT NULL DEFAULT '*',
	`added_on` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	`last_active` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (`id`),
	UNIQUE (`username`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

CREATE TABLE `gc_wordfilters` (
	`id` SERIAL,
	`search` VARCHAR(75) NOT NULL CHECK (search <> ''),
	`change_to` VARCHAR(75) NOT NULL DEFAULT '',
	`boards` VARCHAR(128) NOT NULL DEFAULT '*',
	`regex` BOOLEAN NOT NULL DEFAULT FALSE,
	PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4;

INSERT INTO `gc_announcements` (`subject`,`message`,`poster`) VALUES('subject','message','admin');
INSERT INTO `gc_banlist`
	(`allow_read`,`ip`,`name`,`filename`,`boards`,`staff`,`permaban`,`reason`,`staff_note`)
	VALUES(1,'127.0.0.1','Meanie','badfile.jpg','test,test2','admin',1,'reason here','staff note');
INSERT INTO `gc_boards` VALUES (1,0,'test',0,0,'Testing','Testing, testing, 123','jieofjeio',1,4718592,11,'pipes.css',0,'2021-06-24 04:08:58','Anonymous',0,0,200,0,8192,1,0,0,1);
INSERT INTO `gc_info` VALUES ('version','2.12.0');
INSERT INTO `gc_posts` VALUES (1,1,0,'Name','3GqYIJ3Obs','email@site.com','Subject','message body','message body','df740f13f6c59841743598b2fd9a45c9','162450778130.jpg','60056568.jpg','02a3317c1e16d88e052c1b6c5f181cd2',32174,750,751,199,200,'172.27.0.1','','2021-06-24 04:09:41',0,'0000-00-00 00:00:00','2021-06-24 04:09:41',0,0,0);
INSERT INTO `gc_reports` (`board`,`postid`,`ip`,`reason`,`cleared`) VALUES('test',1,'127.0.0.1','bad post pls delet',0);
INSERT INTO `gc_sections` VALUES (1,0,0,'Main','main');
INSERT INTO `gc_staff` VALUES (1,'admin','$2a$04$L8aNP6T4IAENeg6YzEI/EOG4JbotSTtC7TC.2rQu/z6aFixOu7c52',3,'*','2021-06-24 04:06:37','2021-06-24 04:08:12');

GRANT ALL PRIVILEGES ON gochan_pre2021_db.* TO 'gochan'@'%';
FLUSH PRIVILEGES;
