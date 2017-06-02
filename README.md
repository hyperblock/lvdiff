# lvbackup

A tool to backup and restore LVM2 thinly-provisioned volumes.

This tool can do full backup and incremental for thin volume. In full backup mode, lvbackup only saves the blocks used by the thin volume to minimize the backup size. In incremental backup mode, lvbackup detects the created/changed/deleted blocks from source volume to target volume. 

All the data, which are written to standard output, can be compressed and saved as local file.

The volumes can be restored from incremental backup file, or multiple continus incremental backup files.

The incremental backup method is inspired by [lvmsync](https://github.com/mpalmer/lvmsync). 

The sub commands is inspired by zfs send/recv.

## Usage (__NEED RUN AS ROOT__)

Create thin snapshot for thin volumes as usual (it's better to freeze the file system before creating snapshot

    lvcreate -s -n {SNAP_NAME} {VG_NAME}/{LV_NAME}
    
You can create incremental backup:
  
    lvbackup send -v {VG_NAME} -l {SNAP_NAME} -i {OLD_SNAP_NAME} --head {HEADER_FILE} -o {OUTPUT_FILE}
	
	eg. lvbackup send -v vg001 -l sp001 -i vol0 --head header -o backup_sp001_0

To restore the volume from backup, you need have the old volume. Then, run recv subcommand: 

    lvbackup recv -v {VG_NAME} -p {POOL_NAME} -l {LV_NAME} -i {BACKUP_FILE}

Please note that the chunk size of thin pool for restoring must be equal to that in the backup files.
