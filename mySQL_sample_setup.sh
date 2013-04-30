#!/usr/bin/env sh
mysql -e "CREATE DATABASE sample_database;"
mysql -e "CREATE TABLE IF NOT EXISTS  sample_database.users_main 
  (
      id              INT(10) UNSIGNED NOT NULL auto_increment,
      uploaded        BIGINT(20) UNSIGNED NOT NULL DEFAULT '0',
      downloaded      BIGINT(20) UNSIGNED NOT NULL DEFAULT '0',
      enabled         ENUM('0', '1', '2') NOT NULL DEFAULT '0',
      torrent_pass    CHAR(32) NOT NULL,
      slots           INT(10) NOT NULL DEFAULT '-1',
      rawup           BIGINT(20) NOT NULL,
      rawdl           BIGINT(20) NOT NULL,
      downmultiplier  FLOAT NOT NULL DEFAULT '1',
      upmultiplier    FLOAT NOT NULL DEFAULT '1',
     PRIMARY KEY ( id ),
     KEY  uploaded  ( uploaded ),
     KEY  downloaded  ( downloaded ),
     KEY  enabled  ( enabled ),
     KEY  torrent_pass  ( torrent_pass )
  )
engine=innodb
DEFAULT charset=utf8;"

mysql -e "CREATE TABLE IF NOT EXISTS  sample_database.torrents 
  (
      id              INT(10) NOT NULL auto_increment,
      info_hash       BLOB NOT NULL,
      leechers        INT(6) NOT NULL DEFAULT '0',
      seeders         INT(6) NOT NULL DEFAULT '0',
      last_action     INT(11) NOT NULL DEFAULT '0',
      freetorrent     ENUM('0', '1') NOT NULL DEFAULT '0',
      downmultiplier  FLOAT NOT NULL DEFAULT '1',
      upmultiplier    FLOAT NOT NULL DEFAULT '1',
      status          INT(11) NOT NULL DEFAULT '0',
      snatched        INT(11) NOT NULL DEFAULT '0',  
     PRIMARY KEY ( id ),
     UNIQUE KEY  infohash  ( info_hash(40) ),
     KEY  last_action  ( last_action )
  )
engine=innodb
DEFAULT charset=utf8;" 

mysql -e "CREATE TABLE IF NOT EXISTS  sample_database.xbt_client_whitelist 
  (
      id       INT(10) UNSIGNED NOT NULL auto_increment,
      peer_id  VARCHAR(20) DEFAULT NULL,
      vstring  VARCHAR(200) DEFAULT '',
      notes    VARCHAR(1000) DEFAULT NULL,
     PRIMARY KEY ( id ),
     UNIQUE KEY  peer_id  ( peer_id )
  )
engine=innodb
DEFAULT charset=utf8; "
 
mysql -e "CREATE TABLE IF NOT EXISTS sample_database.mod_core
  (
     mod_setting varchar(20),
     mod_option varchar(20)
  )
engine=innodb
DEFAULT charset=utf8;"

mysql -e "CREATE TABLE IF NOT EXISTS sample_database.transfer_history 
  (
      uid            INT(11) NOT NULL DEFAULT '0',
      fid            INT(11) NOT NULL DEFAULT '0',
      uploaded       BIGINT(20) NOT NULL DEFAULT '0',
      downloaded     BIGINT(20) NOT NULL DEFAULT '0',
      connectable    ENUM('0', '1') NOT NULL DEFAULT '0',
      seeding        ENUM('0', '1') NOT NULL DEFAULT '0',
      seedtime       INT(30) NOT NULL DEFAULT '0',
      hnr            ENUM('0', '1', '2') NOT NULL DEFAULT '0',
      hnrsettime     DATETIME NOT NULL DEFAULT '0000-00-00 00:00:00',
      remaining      BIGINT(20) NOT NULL DEFAULT '0',
      active         ENUM('0', '1') NOT NULL DEFAULT '0',
      starttime      INT(11) NOT NULL DEFAULT '0',
      last_announce  INT(11) NOT NULL DEFAULT '0',
      snatched       INT(11) NOT NULL DEFAULT '0',
      snatched_time  INT(11) DEFAULT '0',
     PRIMARY KEY ( uid ,  fid ),
     KEY  uid  ( uid ),
     KEY  fid  ( fid )
  )
engine=innodb
DEFAULT charset=utf8; " 

