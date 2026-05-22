# LMS MySQL dump for local LK

1. Place full dump at workspace root as `dump.sql`, or copy slice here.
2. Build `01-openedx-only.sql` (openedx only, FK checks off):

```bash
{ echo "SET GLOBAL FOREIGN_KEY_CHECKS=0; SET SESSION FOREIGN_KEY_CHECKS=0;";
  tail -n +1073 /path/to/dump.sql;
  echo "SET GLOBAL FOREIGN_KEY_CHECKS=1;"; } > 01-openedx-only.sql
```

3. Do **not** use symlinks — Docker init cannot follow them.
4. Start: `docker compose -f docker-compose.lms_mysql.yml up -d`
5. After healthy: `GRANT SELECT ON openedx.* TO 'lk_readonly'@'%';` (root password `lms_root_change_me`)
