# Systemd Configuration

## Files

```bash
root@tribenet:/etc/systemd/system# pwd
/etc/systemd/system

root@tribenet:/etc/systemd/system# ll otto*
-rw-r--r-- 1 root root 241 Oct  4 06:18 ottomap.service
-rw-r--r-- 1 root root 188 Oct 13 22:33 ottomap.timer
-rw-r--r-- 1 root root 421 Sep 30 22:06 ottoapp.service

root@tribenet:/var/www/ottomap.mdhenderson.com/bin# pwd
/var/www/ottomap.mdhenderson.com/bin

root@tribenet:/var/www/ottomap.mdhenderson.com/bin# ll
total 24408
drwxrwxr-x 2 mdhender mdhender     4096 Oct 13 22:24 ./
drwxr-xr-x 6 mdhender mdhender     4096 Oct 13 21:57 ../
-rwxr-xr-x 1 mdhender mdhender  6957655 Oct 13 22:11 ottomap*
-rwxr-xr-x 1 mdhender mdhender      439 Oct  4 06:07 ottomap.service.sh*
-rwxr-xr-x 1 mdhender mdhender     5561 Oct 13 22:24 ottomap.sh*
-rwxr-xr-x 1 mdhender mdhender 18013459 Oct 13 21:57 ottoapp*
```

## Install

```bash
root@tribenet:/etc/systemd/system# systemctl daemon-reload

root@tribenet:/etc/systemd/system# systemctl enable ottoapp.service

root@tribenet:/etc/systemd/system# systemctl status ottoapp.service
● ottoapp.service - Otto Web server
     Loaded: loaded (/etc/systemd/system/ottoapp.service; enabled; preset: enabled)
     Active: active (running) since Sun 2024-10-13 22:46:52 UTC; 4h 37min ago
   Main PID: 1778 (ottoapp)
      Tasks: 7 (limit: 1112)
     Memory: 3.7M (peak: 6.2M)
        CPU: 153ms
     CGroup: /system.slice/ottoapp.service
             └─1778 /var/www/ottomap.mdhenderson.com/bin/ottoapp serve --database /home/ottoapp/data/ottoapp.db

root@tribenet:/etc/systemd/system# systemctl enable ottomap.service
○ ottomap.service - Run ottomap every 1 minutes
     Loaded: loaded (/etc/systemd/system/ottomap.service; static)
     Active: inactive (dead) since Mon 2024-10-14 03:25:28 UTC; 3s ago
   Duration: 57ms
TriggeredBy: ● ottomap.timer
    Process: 8709 ExecStart=/var/www/ottomap.mdhenderson.com/bin/ottomap.service.sh (code=exited, status=0/SUCCESS)
   Main PID: 8709 (code=exited, status=0/SUCCESS)
        CPU: 35ms

root@tribenet:/etc/systemd/system# systemctl enable ottomap.timer
● ottomap.timer - Run ottomap.service.sh 30 seconds after it finishes
     Loaded: loaded (/etc/systemd/system/ottomap.timer; enabled; preset: enabled)
     Active: active (waiting) since Sun 2024-10-13 22:33:51 UTC; 4h 52min ago
    Trigger: Mon 2024-10-14 03:25:58 UTC; 2s left
   Triggers: ● ottomap.service
```

## Monitor

```bash
root@tribenet:/etc/systemd/system# journalctl -f -u ottoapp.service

root@tribenet:/etc/systemd/system# journalctl -f -u ottomap.service

root@tribenet:/etc/systemd/system# journalctl -f -u ottomap.timer
```

