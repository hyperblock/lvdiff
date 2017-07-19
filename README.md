# lvdiff

A pair of tools ( __lvdiff/lvpatch__ ) to backup and restore LVM2 thinly-provisioned volumes.


## Usage (__NEED RUN AS ROOT__)

### lvdiff
lvdiff is a tool to backup LVM2 thinly-provisioned volumes, will dump the thin volume $volume\_A's incremental blocks from $volume\_B

```
Usage:
  lvdiff <volume_A> <volume_B> [flags]

Flags:
  -d, -- int32             checksum detect level. range: 0-3 
							 0 means no checksum, 
							 1 means only check head block, 
							 2 means random check, 
							 3 means scan all data blocks. (default 2)
  -h, --help       help for lvdiff
      --meta	   set metadata (format as '$key:$value').
 	           to set one more meta, use --meta aa:aa --meta bb:bb

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

# Example

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

## lvpatch
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
  
## NOTE
Use command __lvs__ to check current volumes in your computer. If an volume is inactive, use command __lvchange -ay -K [volume path]__ to active it before mount.

# lvdiff

__lvdiff/lvpatch__ 是一组用于对LVM自动精简配置(Thin Provisioning)的逻辑卷进行差异比较和补丁的工具。使用 __lvdiff__ 可以将两个逻辑卷的差异形成二进制文件导出,再通过 __lvpatch__ 可将差异文件拼凑至另外的逻辑卷之上，实现逻辑卷的迁移。


## Usage (__NEED RUN AS ROOT__)

### lvdiff
__lvdiff__ 用于将指定的两个逻辑卷A和B之间的差异数据块导出成二进制文件。
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
__lvpatch__ 用于将lvdiff导出的结果拼凑到另一个逻辑卷之上。当该逻辑卷与原逻辑卷B的数据一致时，可实现逻辑卷A的异地恢复。

```
Usage:
  lvpatch [flags]

Flags:
  -h, --help             help for lvpatch
  -l, --lv string        logical volume
  -g, --lvgroup string   volume group
      --no-base-check    patch volume into base without calculate checksum.
```

请注意，不同逻辑卷所在的精简池的chunk size需要相同才能进行正常恢复。

# Example

## lvdiff 
这一部分为使用 __lvdiff__ 导出差异文件的示例。

1. 创建一个虚拟的块设备 
```
$ dd if=/dev/zero of=1G.disk.0 bs=1M count=1024
$ losetup /dev/loop0 1G.disk.0   //assume loop0 is an unused device
```
2. 建立逻辑卷vol0
```
$ pvcreate /dev/loop0
$ vgcreate vg0 /dev/loop0
$ lvcreate --thinpool vg0/pool0 -l 100%FREE /dev/vg0
$ lvcreate -T vg1/pool0 -V 500M vol0
```
3. 建立 vol0 的快照 sp0
```
$ lvcreate -s -n sp0 /dev/vol0 //now we have an base volume  without any data.
```
4. 向 vol0 中写入数据 
```
$ mkfs.ext4 /dev/vg0/vol0
$ mount /dev/vg0/vol0 /mnt/vg0/vol0
$ echo "hello world" > /mnt/vg0/vol0/hello
$ ..... //write some files to /dev/vg0/vol0
```
5. 导出 vol0 与 sp0 之间的差异数据块，同时添加一些导出信息:
```
$ lvdiff -g vg0 --pair 'Author:BigVan (alpc_metic@live.com)' --pair 'Message: hello world' sp0 vol0 > test.diff
```
vol0 和 sp0 之间的差异数据将被保存为 test.diff . 同时，该文件的 SHA1 结果将被输出至 Stderr.

## lvpatch
这一部分将把 test.diff 拼接到与上一节中vg0/sp0一致的另一逻辑卷至上，实现 vg0/vol0 的异地恢复。
1. 创建一个虚拟的块设备 
```
$ dd if=/dev/zero of=1G.disk.1 bs=1M count=1024
$ losetup /dev/loop1 1G.disk.1   //assume loop1 is an unused device
```
2. 创建逻辑卷 sp0
```
$ pvcreate /dev/loop1
$ vgcreate vg1 /dev/loop1
$ lvcreate --thinpool vg1/pool0 -l 100%FREE /dev/vg1
$ lvcreate -T vg1/pool0 -V 500M sp0 
```
3. 将 test.diff 拼凑到 sp0 之上
```
$ cat test.diff | sudo ./lvpatch -g vg1 -l sp0
``` 
 lvpatch将在sp0的基础上，通过写入 test.diff 到特定位置，实现vg0/vol0 在vg1上的恢复。
  
## 注意
使用命令 __lvs__ 用于列出当前机器上存在的逻辑卷. 如果某一逻辑卷在挂在前未激活, 需要通过__lvchange -ay -K [volume path]__ 命令去激活该卷。
