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


### lvpatch
lvpatch is a tool to patch volume's diff file (a set of volume's change blocks) to another thin-volume.  

```
Usage:
  lvpatch [flags]

Flags:
  -h, --help             help for lvpatch
  -l, --lv string        logical volume
  -g, --lvgroup string   volume group
      --no-base-check    patch volume into base without calculate checksum.
```

Please note that the chunk size of thin pool for restoring must be equal to that in the backup files.

#Example

## lvdiff 
In this section, we  will use __lvdiff__ to create an diff-block file of thin volume.

1. Create a device
```
$ dd if=/dev/zero of=1G.disk.0 bs=1M count=1024
$ losetup /dev/loop0 1G.disk.0   //assume loop0 is an unused device
```
2. Create Thin volume __vol0__
```
$ pvcreate /dev/loop0
$ vgcreate vg0 /dev/loop0
$ lvcreate --thinpool vg0/pool0 -l 100%FREE /dev/vg0
$ lvcreate -T vg1/pool0 -V 500M vol0
```
3. Create an snapshot __sp0__ of __vol0__
```
$ lvcreate -s -n sp0 /dev/vol0 //now we have an base volume  without any data.
```
4. Write some data into __vol0__
```
$ mkfs.ext4 /dev/vg0/vol0
$ mount /dev/vg0/vol0 /mnt/vg0/vol0
$ echo "hello world" > /mnt/vg0/vol0/hello
$ ..... //write some files to /dev/vg0/vol0
```
5. dump diff file:
```
$ lvdiff -g vg0 --pair 'Author:BigVan (alpc_metic@live.com)' --pair 'Message: hello world' sp0 vol0 > test.diff
```
lvdiff will dump the different blocks between __vol0__ and __sp0__ and saved as __test.diff__. And SHA1 code will be shown in Stderr.

##lvpatch
In this  section, we will patch __test.diff__ to an base volume which is identical with __vg0/sp0__ .
1. Create a device
```
$ dd if=/dev/zero of=1G.disk.1 bs=1M count=1024
$ losetup /dev/loop1 1G.disk.1   //assume loop1 is an unused device
```
2. Create Thin volume __sp0__
```
$ pvcreate /dev/loop1
$ vgcreate vg1 /dev/loop1
$ lvcreate --thinpool vg1/pool0 -l 100%FREE /dev/vg1
$ lvcreate -T vg1/pool0 -V 500M sp0 
```
3. Patch __test.diff__ to __vg1/sp0__
```
$ cat test.diff | sudo ./lvpatch -g vg1 -l sp0
``` 
  It will restore thin snapshot volume /dev/vg1/vol0
  
##NOTE
Use command __lvs__ to check current volumes in your computer. If an volume is inactive, use command __lvchange -ay -K [volume path]__ to active it before mount.

