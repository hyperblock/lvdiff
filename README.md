# lvdiff

A pair of tools ( __lvdiff/lvpatch__ ) to backup and restore LVM2 thinly-provisioned volumes.

## Usage (__NEED RUN AS ROOT__)

Create a thin snapshot:  
    
	lvcreate -n -s <SNAP_SHOT> <VG_NAME>/<VOL_NAME>

### lvdiff
lvdiff is a tool to backup LVM2 thinly-provisioned volumes, will dump the thin volume $volume's incremental block from $backing-volume  

```
Usage:
  lvdiff <volume> <backing-volume> [flags]

Flags:
  -d, -- int32             checksum detect level. range: 0-3 
							 0 means no checksum, 
							 1 means only check head block, 
							 2 means random check, 
							 3 means scan all data blocks. (default 2)
  -h, --help       help for lvdiff
      --pair	   set key-value pair (format as '$key:$value').
  -p, --pool       thin volume pool.
```

__eg.__  
```
For a volume group:

    vg001
      - thin_pool0
        - vol0
        - snap001
        
run command:  

   sudo ./lvdiff -g vg001 --pair 'Author:BigVan (alpc_metic@live.com)' --pair 'Message: hello world'sp001 vol0 > sp001_vol0.diff
```
   It will dump the different blocks between 'sp001' and 'vol0' and save as 'sp001\_vol0.diff'. And SHA1 code will be shown in Stderr.


### lvpatch
lvpatch is a tool to patch volume's diff file (a set of volume's change blocks) to another thin-volume.  

```
Usage:
  lvpatch <input_diff_file> [flags]

Flags:
  -h, --help             help for lvpatch
  -l, --lv string        logical volume
  -g, --lvgroup string   volume group
      --no-base-check    patch volume into base without calculate checksum.
  -p, --pool string      thin pool
```

__eg.__  
```
For a volume group:  

    vg001
     - thin_pool0
       - vol0

run command:
   
    sudo ./lvpatch -g vg001 -p thin_pool0 -l vol0 sp001_vol0.diff
``` 
   It will restore thin snapshot volume /dev/vg001/sp001
  

Please note that the chunk size of thin pool for restoring must be equal to that in the backup files.
