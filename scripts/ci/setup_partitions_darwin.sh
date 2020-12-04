#!/bin/bash

# Exit immediately on failure.
set -e

# Create and mount a FAT32 partition.
# NOTE: There seems to be a lower bound on the size here, though I haven't
# probed around enough to see what it is (20 MB is too small), and I'm not sure
# if it applies to the other filesystems.
hdiutil create -megabytes 50 -fs "MS-DOS FAT32" -volname FAT32ROOT -o fat32image.dmg
hdiutil attach fat32image.dmg

# Create and mount an HFS+ parition.
hdiutil create -megabytes 50 -fs "HFS+" -volname "HFSRoot" -o hfsimage.dmg
hdiutil attach hfsimage.dmg

# Create and mount an APFS partition.
hdiutil create -megabytes 50 -fs "APFS" -volname "APFSRoot" -o apfsimage.dmg
hdiutil attach apfsimage.dmg

# Create a second FAT32 partition and mount it inside the APFS mount.
# NOTE: I originally tried the volume name "FAT32SUBROOT", but FAT32 has severe
# volume name length restrictions, and hdiutil won't notify you if the name is
# too long - it will just use "NO NAME".
hdiutil create -megabytes 50 -fs "MS-DOS FAT32" -volname FAT32SUB -o fat32subimage.dmg
hdiutil attach -mountroot "/Volumes/APFSRoot" fat32subimage.dmg
