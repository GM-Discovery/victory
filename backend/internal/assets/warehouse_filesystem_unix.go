//go:build !windows

package assets

import "syscall"

func loadWarehouseFilesystemStats(storageRoot string) (warehouseFilesystemStats, error) {
	root := resolveStorageRoot(storageRoot)

	var stat syscall.Statfs_t
	if err := syscall.Statfs(root, &stat); err != nil {
		return warehouseFilesystemStats{}, err
	}

	blockSize := int64(stat.Bsize)
	totalBytes := int64(stat.Blocks) * blockSize
	freeBytes := int64(stat.Bfree) * blockSize
	availableBytes := int64(stat.Bavail) * blockSize
	usedBytes := totalBytes - freeBytes
	usableUploadBytes := availableBytes - WarehousePhysicalReserveBytes
	if usableUploadBytes < 0 {
		usableUploadBytes = 0
	}

	return warehouseFilesystemStats{
		Path:              root,
		TotalBytes:        totalBytes,
		FreeBytes:         freeBytes,
		AvailableBytes:    availableBytes,
		UsedBytes:         usedBytes,
		ReserveBytes:      WarehousePhysicalReserveBytes,
		UsableUploadBytes: usableUploadBytes,
	}, nil
}
