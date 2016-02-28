### All tools is alpha (work bad, but work)

# Unpacker (GoW1 + GoW2)
Tool for unpaking part\*.pak files using info from *GODOFWAR.TOC*

Autodetecting version of GoW

**GoW1 Tested only on 1-dvd version with part1.pak file**

**GoW2 Tested only on 1-dvd version with part1-5.pak files**

Usage: *./unpacker path_to_game_folder [path_to_store_files]*

path_to_store_files default is *path_to_game_folder + /pack*

Formats in archive:

| Format | Info |
|-------:|:-----|
| PSS/PSW | mpeg videos (without sound). PSS without headers (or not)|
| WAD | game archives, can use [Wadreader](#Wadreader) to unpack |
| VAG/VA1-5 | VAGp ADPCM sounds (depended on language) |
| VPK | RAW ADPCM music |
| TXT | SANITY.TXT used for check archive validity |

After unpaking, summary size of all files being lower then size of archive. This is because, archive dublicate files for faster access on disk.

# Wadreader (GoW1 + GoW2)
Tool for unpaking *.wad files. Probably not unpack all.

Autodetecting version of GoW

Usage: *./wadreader path_to_wad_file [path_to_store_files]*

path_to_store_files default is *path_to_wad_file + _unpacked*

# gfx2img (GoW1)
Convert gfx + pal textures to png image

Usage: *./gfx2img path_to_gfx_file [--pal custom_pallete_file] [--output_file_name]*
