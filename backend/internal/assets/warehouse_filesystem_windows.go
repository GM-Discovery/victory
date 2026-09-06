//go:build windows

package assets

import "golang.org/x/sys/windows"

// Kernel 100 (windows-native): Windows has no statfs(2) equivalent --
// GetDiskFreeSpaceEx is the documented Win32 API for exactly this, wrapped
// by the same golang.org/x/sys/windows module already an indirect
// dependency of this project.
func loadWarehouseFilesystemStats(storageRoot string) (warehouseFilesystemStats, error) {
	root := resolveStorageRoot(storageRoot)

	rootPtr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return warehouseFilesystemStats{}, err
	}

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(rootPtr, &freeBytesAvailable, &totalBytes, &totalFreeBytes); err != nil {
		return warehouseFilesystemStats{}, err
	}

	usedBytes := int64(totalBytes) - int64(totalFreeBytes)
	usableUploadBytes := int64(freeBytesAvailable) - WarehousePhysicalReserveBytes
	if usableUploadBytes < 0 {
		usableUploadBytes = 0
	}

	return warehouseFilesystemStats{
		Path:              root,
		TotalBytes:        int64(totalBytes),
		FreeBytes:         int64(totalFreeBytes),
		AvailableBytes:    int64(freeBytesAvailable),
		UsedBytes:         usedBytes,
		ReserveBytes:      WarehousePhysicalReserveBytes,
		UsableUploadBytes: usableUploadBytes,
	}, nil
}
