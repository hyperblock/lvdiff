all: lvbackup

lvbackup: *.go
	# Only build for linux as LVM2 only works for linux
	GOOS=linux go build
