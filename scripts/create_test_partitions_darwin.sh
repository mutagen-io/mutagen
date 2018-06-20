#!/bin/bash

# Create and mount a FAT32 partition.
# NOTE: There seems to be a lower bound on the size here, though I haven't
# probed around enough to see what it is (20 MB is too small), and I'm not sure
# if it applies to the other filesystems.
hdiutil create -megabytes 50 -fs "MS-DOS FAT32" -volname FAT32ROOT -o fat32image.dmg || exit $?
hdiutil attach fat32image.dmg || exit $?
export MUTAGEN_TEST_FAT32_ROOT="/Volumes/FAT32ROOT"

# Create and mount an HFS+ parition.
hdiutil create -megabytes 50 -fs "HFS+" -volname "HFSRoot" -o hfsimage.dmg || exit $?
hdiutil attach hfsimage.dmg || exit $?
export MUTAGEN_TEST_HFS_ROOT="/Volumes/HFSRoot"

# Create and mount an APFS partition.
hdiutil create -megabytes 50 -fs "APFS" -volname "APFSRoot" -o apfsimage.dmg || exit $?
hdiutil attach apfsimage.dmg || exit $?
export MUTAGEN_TEST_APFS_ROOT="/Volumes/APFSRoot"

# Create a second FAT32 partition and mount it inside the APFS mount.
# NOTE: I originally tried the volume name "FAT32SUBROOT", but FAT32 has severe
# volume name length restrictions, and hdiutil will fail to notify you if the
# name is too long and will just use "NO NAME".
hdiutil create -megabytes 50 -fs "MS-DOS FAT32" -volname FAT32SUB -o fat32subimage.dmg || exit $?
hdiutil attach -mountroot "${MUTAGEN_TEST_APFS_ROOT}" fat32subimage.dmg || exit $?
export MUTAGEN_TEST_FAT32_SUBROOT="${MUTAGEN_TEST_APFS_ROOT}/FAT32SUB"
