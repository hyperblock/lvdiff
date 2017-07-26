# lvdiff Project 
_https://github.com/hyperblock/lvdiff_

A pair of tools ( __lvdiff/lvpatch__ ) to backup and restore LVM2 thinly-provisioned volumes.


## Usage (__NEED RUN AS ROOT__)

### lvdiff
lvdiff is a tool to dump the differential blocks of two __LVM2 thinly-provisioned volumes__.

The format of dump file is called __HyperLayer__. ( http://www.hyperblock.org/2017/06/16/hyperlayer/ )

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
      --meta       set metadata (format as '$key:$value').
 	           To set one more meta, use --meta aa:aa --meta bb:bb

```


### lvpatch
lvpatch is a tool to patch volume's diff file (a set of volume's change blocks) to another thin-volume.  

```
Usage:
  lvpatch <new_volume_name> [flags]

Flags:
  -h, --help                help for lvpatch
  -l, --lvbase string       base logical volume
  -g, --lvgroup string      volume group
      --no-base-check       patch volume into base without calculate checksum.
```


# Example

## lvdiff 
In this section, we  will use __lvdiff__ to create an diff-block file of thin volume.

Assuming there is a thin-volume pool __'pool0'__ in volume group __'vg0'__

1. Create an snapshot __sp0__ of __vol0__
```
$ lvcreate -s -n sp0 /dev/vol0 
```
2. Write some data into __vol0__
```
$ mount /dev/vg0/vol0 /mnt/vg0/vol0
$ echo "hello world" > /mnt/vg0/vol0/hello
$ ..... //write some files to /dev/vg0/vol0
```
3. dump diff file:
```
$ lvdiff -g vg0 --meta 'Author:BigVan (alpc_metic@live.com)' --meta 'Message: hello world' sp0 vol0 > test.diff
```
lvdiff will dump the different blocks between __vol0__ and __sp0__ and saved as __test.diff__. And SHA1 code will be shown in Stderr.

## lvpatch
In this  section, we will patch __test.diff__ to a base volume __'vg1/sp0'__ which is identical with __vg0/sp0__ . 

3. Patch __test.diff__ to __vg1/sp0__
```
$ cat test.diff | sudo ./lvpatch -g vg1 -l sp0 vol0_new
``` 
  It will restore thin volume __/dev/vg1/vol0_new__
  
## NOTE
Use command __lvs__ to check current volumes in your computer. If an volume is inactive, use command __lvchange -ay -K [volume path]__ to active it before mount.

# lvdiff
_https://github.com/hyperblock/lvdiff_

__lvdiff/lvpatch__ 是一组用于对LVM自动精简配置(Thin Provisioning)的逻辑卷进行差异比较和补丁的工具。使用 __lvdiff__ 可以将两个逻辑卷的差异数据块形成二进制文件导出,再通过 __lvpatch__ 可将差异文件拼凑至另外的逻辑卷之上，实现逻辑卷的迁移。

通过lvdiff导出的文件格式称作 __HyperLayer__ ( http://www.hyperblock.org/2017/06/16/hyperlayer/ )

## Usage (__NEED RUN AS ROOT__)

### lvdiff
__lvdiff__ 用于将指定的两个逻辑卷A和B之间的差异数据块导出成二进制文件。
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
      --meta       set metadata (format as '$key:$value').
 	           To set one more meta, use --meta aa:aa --meta bb:bb

```
### lvpatch
__lvpatch__ 用于将lvdiff导出的结果拼凑到另一个 __逻辑卷base__ 之上并形成新卷。当 __逻辑卷base__ 与 __原逻辑卷B__ 的数据一致时，可实现逻辑卷A的异地恢复。

```
Usage:
  lvpatch <new_volume_name> [flags]

Flags:
  -h, --help                help for lvpatch
  -l, --lvbase string       base logical volume
  -g, --lvgroup string      volume group
      --no-base-check       patch volume into base without calculate checksum.
```

# Example

## lvdiff 
这一部分为使用 __lvdiff__ 的示例。
假设在逻辑卷组 __'vg0'__ 中有精简配置池 __'pool0'__

1. 建立 vol0 的快照 sp0
```
$ lvcreate -s -n sp0 /dev/vol0
```
2. 向 vol0 中写入数据 
```
$ mount /dev/vg0/vol0 /mnt/vg0/vol0
$ echo "hello world" > /mnt/vg0/vol0/hello
$ ..... //write some files to /dev/vg0/vol0
```
3. 导出 vol0 与 sp0 之间的差异数据块，同时添加一些导出信息:
```
$ lvdiff -g vg0 --meta 'Author:BigVan (alpc_metic@live.com)' --meta 'Message: hello world' sp0 vol0 > test.diff
```
vol0 和 sp0 之间的差异数据将被保存为 test.diff . 同时，该文件的 SHA1 结果将被输出至 Stderr.

## lvpatch
这一部分将把 test.diff 拼接到与上一节中 __vg0/sp0__ 一致的另一逻辑卷 __'vg1/sp0'__ 上，实现 vg0/vol0 的异地恢复。
3. 将 test.diff 拼凑到 sp0 之上
```
$ cat test.diff | sudo ./lvpatch -g vg1 -l sp0 vol0_new
``` 
 新卷的名字为 __vg1/vol0_new__.
  
## 注意
使用命令 __lvs__ 用于列出当前机器上存在的逻辑卷. 如果某一逻辑卷在挂在前未激活, 需要通过 __lvchange -ay -K [volume path]__ 命令去激活该卷。

