//go:build windows

package acl

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	advapi32                    = windows.NewLazySystemDLL("advapi32.dll")
	procGetNamedSecurityInfoW   = advapi32.NewProc("GetNamedSecurityInfoW")
	procSetNamedSecurityInfoW   = advapi32.NewProc("SetNamedSecurityInfoW")
	procConvertSecDescriptorToStringSecDescriptor = advapi32.NewProc("ConvertSecurityDescriptorToStringSecurityDescriptorW")
	procConvertStringSecDescriptorToSecDescriptor = advapi32.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
)

const (
	seFileObject          = 1  // SE_FILE_OBJECT
	daclSecurityInfo      = 4  // DACL_SECURITY_INFORMATION
	ownerSecurityInfo     = 1  // OWNER_SECURITY_INFORMATION
	groupSecurityInfo     = 2  // GROUP_SECURITY_INFORMATION
	sacl_security_info    = 8  // SACL_SECURITY_INFORMATION (requires SeSecurityPrivilege)
	stdSecurityInfo       = ownerSecurityInfo | groupSecurityInfo | daclSecurityInfo
	sddlRevision1         = 1
)

// GetSDDL returns the Windows security descriptor of path in SDDL string format.
// SDDL encodes Owner, Group, DACL (and optionally SACL) as a portable string.
func GetSDDL(path string) (string, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", fmt.Errorf("acl: encode path: %w", err)
	}

	var pSD uintptr
	r, _, err := procGetNamedSecurityInfoW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		seFileObject,
		stdSecurityInfo,
		0, 0, 0, 0,
		uintptr(unsafe.Pointer(&pSD)),
	)
	if r != 0 {
		return "", fmt.Errorf("acl: GetNamedSecurityInfo(%q): %w", path, err)
	}
	defer windows.LocalFree(windows.Handle(pSD))

	var sddlPtr uintptr
	r, _, err = procConvertSecDescriptorToStringSecDescriptor.Call(
		pSD,
		sddlRevision1,
		stdSecurityInfo,
		uintptr(unsafe.Pointer(&sddlPtr)),
		0,
	)
	if r == 0 {
		return "", fmt.Errorf("acl: ConvertSDToString(%q): %w", path, err)
	}
	defer windows.LocalFree(windows.Handle(sddlPtr))

	sddl := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(sddlPtr)))
	return sddl, nil
}

// SetSDDL applies an SDDL security descriptor to path.
func SetSDDL(path, sddl string) error {
	sddlPtr, err := syscall.UTF16PtrFromString(sddl)
	if err != nil {
		return fmt.Errorf("acl: encode SDDL: %w", err)
	}

	var pSD uintptr
	var pSDSize uint32
	r, _, e := procConvertStringSecDescriptorToSecDescriptor.Call(
		uintptr(unsafe.Pointer(sddlPtr)),
		sddlRevision1,
		uintptr(unsafe.Pointer(&pSD)),
		uintptr(unsafe.Pointer(&pSDSize)),
	)
	if r == 0 {
		return fmt.Errorf("acl: ConvertStringToSD(%q): %w", path, e)
	}
	defer windows.LocalFree(windows.Handle(pSD))

	// Parse out owner, group, DACL pointers from the self-relative SD
	var owner, group, dacl uintptr
	var ownerDefaulted, groupDefaulted, daclPresent, daclDefaulted bool

	sd := (*windows.SECURITY_DESCRIPTOR)(unsafe.Pointer(pSD))
	_ = sd // Use windows API calls instead of direct struct access for safety

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("acl: encode path: %w", err)
	}

	// GetSecurityDescriptorOwner/Group/Dacl to extract pointers
	procGetOwner := advapi32.NewProc("GetSecurityDescriptorOwner")
	procGetGroup := advapi32.NewProc("GetSecurityDescriptorGroup")
	procGetDacl  := advapi32.NewProc("GetSecurityDescriptorDacl")

	var ownerDef, groupDef int32
	procGetOwner.Call(pSD, uintptr(unsafe.Pointer(&owner)), uintptr(unsafe.Pointer(&ownerDef)))
	procGetGroup.Call(pSD, uintptr(unsafe.Pointer(&group)), uintptr(unsafe.Pointer(&groupDef)))

	var daclPres int32
	procGetDacl.Call(pSD, uintptr(unsafe.Pointer(&daclPres)), uintptr(unsafe.Pointer(&dacl)), uintptr(unsafe.Pointer(&daclDefaulted)))

	_ = ownerDefaulted
	_ = groupDefaulted
	_ = daclPresent

	r, _, e = procSetNamedSecurityInfoW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		seFileObject,
		stdSecurityInfo,
		owner,
		group,
		dacl,
		0,
	)
	if r != 0 {
		return fmt.Errorf("acl: SetNamedSecurityInfo(%q): %w", path, e)
	}
	return nil
}
