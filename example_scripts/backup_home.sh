logger The Scheduled Backup of the Home Directories is starting now.
echo BraveSoftware           > exclude.list
echo *.log                  >> exclude.list
echo .nvm/                  >> exclude.list
echo .vscode/               >> exclude.list
echo CacheStorage/*         >> exclude.list
echo .cache/                >> exclude.list
echo node_modules/*         >> exclude.list
echo .local/share/flatpak/  >> exclude.list

notify-send  -u normal "The Scheduled Backup of the Home Directories is starting now." "\nConsider logging off"
rsync -r -t -p -o -g -v --progress -s --exclude-from='exclude.list' /myhome/michael sysadm@bkupserv1:/home/sysadm/backup
if [ "$?" -eq "0" ]
then
  notify-send  -u normal "Backup of the Home Directories has completed" " \nResume normal processing."
  logger Backup of the Home Directories has completed successfully
else
  notify-send  -u critical "Backup of Home Directories has Fail" " \nOperator Intervention is required."
  logger ERROR: Backup of Home Directories has failed
fi
