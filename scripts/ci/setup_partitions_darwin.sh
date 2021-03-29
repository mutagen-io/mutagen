#!/bin/bash

# Exit immediately on failure.
set -e

# TODO: We can actually specify the mount points for these images as part of the
# "hdiutil attach" command, so we don't need to rely on their volume names and
# assumptions about them being mounted in /Volumes. We might want to generate a
# specific testing directory at which to mount these.

# TODO: Write a corresponding script to unmount these volumes. It's not critical
# for CI testing, but it would be nice to have for local testing.

# Create and mount a FAT32 partition.
# NOTE: There seems to be a lower bound on the size here, though I haven't
# probed around enough to see what it is (20 MB is too small), and I'm not sure
# if it applies to the other filesystems. Also, FAT32 has serious volume name
# length limitations and hdiutil won't notify you if the name is too long - it
# will just use "NO NAME" for the volume name and you'll be surprised when it
# doesn't mount where you expect, so it's best to test the volume name you want
# manually before scripting it.
hdiutil create -megabytes 50 -fs "MS-DOS FAT32" -volname FAT32ROOT -o fat32image.dmg
hdiutil attach fat32image.dmg

# Create and mount an HFS+ parition.
hdiutil create -megabytes 50 -fs "HFS+" -volname "HFSRoot" -o hfsimage.dmg
hdiutil attach hfsimage.dmg

# Create and mount an APFS partition.
hdiutil create -megabytes 50 -fs "APFS" -volname "APFSRoot" -o apfsimage.dmg
hdiutil attach apfsimage.dmg

# Create and mount an additional HFS+ partition inside the APFS partition to
# test filesystem boundary crossing.
hdiutil create -megabytes 50 -fs "HFS+" -volname "HFSSub" -o hfssubimage.dmg
hdiutil attach -mountpoint "/Volumes/APFSRoot/HFSSub" hfssubimage.dmg
