# lvbackup

A tool to backup and restore LVM2 thinly-provisioned volumes.

This tool can do full backup and incremental for thin volume. In full backup mode, lvbackup only saves the blocks used by the thin volume to minimize the backup size. In incremental backup mode, lvbackup detects the created/changed/deleted blocks from source volume to target volume. 

All the data, which are written to standard output, can be compressed and saved as local file, or transported to anothing host using network tools, like nc and ssh. 

The volumes can be restored from single full backup file, or multiple continus incremental backup files. As the volume UUID is recorded in the backup, broken incremental backup chain will be detected and reported.

The incremental backup method is inspired by [lvmsync](https://github.com/mpalmer/lvmsync). 

The sub commands is inspired by zfs send/recv.

## Installation

    $ go get -u github.com/yangjian/lvbackup

## Usage

Create thin snapshot for thin volumes as usual (it's better to freeze the file system before creating snapshot

    lvcreate -s -n {SNAP_NAME} {VG_NAME}/{LV_NAME}
    
Create full backup, save it into file:
 
    lvbackup send -v {VG_NAME} -l {SNAP_NAME} > {OUTPUT_FILE}
    
Or send it to another host by network:
 
    lvbackup send -v {VG_NAME} -l {SNAP_NAME} | nc {OTHER_HOST}
    
If there is an old snaphot, you can create incremental backup:
  
    lvbackup send -v {VG_NAME} -l {SNAP_NAME} -i {OLD_SNAP_NAME} > {OUTPUT_FILE}

To check the info of backup file:
    
    lvbackup info {BACKUP_FILE}

To restore the volume from full backup, create volume group and thin pool if they do not exists. Then, run recv subcommand: 

    lvbackup recv -v {VG_NAME} -p {POOL_NAME} -l {LV_NAME} < {BACKUP_FILE}

To restore the volume from backup chain:

    cat {FULL_BACKUP} {DELTA_0} {DELTA_1} ... | \
    lvbackup recv -v {VG_NAME} -p {POOL_NAME} -l {LV_NAME}

Please note that the chunk size of thin pool for restoring must be equal to that in the backup files.

## TODO

* Feature: merge continus incremental backups into single one
* Feature: display backup/restore progress
* Enhance: add unit tests
* Enhance: improve the error message displaying
* Enhance: add document about the format of stream data

