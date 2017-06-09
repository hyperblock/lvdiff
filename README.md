# lvdiff

A pair of tools ( lvdiff/lvpatch ) to backup and restore LVM2 thinly-provisioned volumes.


## Usage (__NEED RUN AS ROOT__)

Create a thin snapshot: lvcreate -n -s <SNAP\_SHOT> <VGNAME>/<VLNAME>

### lvdiff
lvdiff is a tool to backup LVM2 thinly-provisioned volumes, will dump the thin volume $volume's incremental block from $backing-volume

__Usage:__  

    lvdiff <volume> <backing-volume> [flags]

    Flags:
    -h, --help               help for lvdiff  
    -g, --lvgroup string     volume group.  
        --pair stringArray   set key-value pair (format as '$key:$value').  
    -p, --pool string        thin volume pool.

__eg.  __  
For a volume group:

    vg001
      - thin_pool0
        - vol0
        - snap001
   __run command:__  
     
    sudo ./lvdiff -g vg001 -p thin_pool0 --pair 'Author:BigVan (alpc_metic@live.com)' sp001 vol0 > sp001_vol0.diff

   It will dump the different blocks between 'sp001' and 'vol0' and save as 'sp001\_vol0.diff'. And SHA1 code will be shown in Stderr.
    

### lvpatch
lvpatch is a tool to patch volume's diff file (a set of volume's change blocks) to another thin-volume.

__Usage:__

    lvpatch <input_diff_file> [flags]
    
    Flags:
      -h, --help             help for lvpatch
      -l, --lv string        logical volume
      -g, --lvgroup string   volume group
      -p, --pool string      thin pool

__eg.__  
For a volume group:  

    vg001
     - thin_pool0
       - vol0

   __run command:__  
   
    sudo ./lvpatch -g vg001 -p thin_pool0 -l vol0 sp001_vol0.diff

   It will restore thin snapshot volume /dev/vg001/sp001
   

Please note that the chunk size of thin pool for restoring must be equal to that in the backup files.
