### All tools is alpha (work bad, but work)

# Unpacker
Tool for unpaking part\*.pak files using info from *GODOFWAR.TOC*

**Tested only on 1-dvd version with solo part1.pak file**

Usage: *./unpacker path_to_game_folder [path_to_store_files]*

path_to_store_files default is *path_to_game_folder + /pack*

# Wadreader
Tool for unpaking *.wad files. Probably not unpack all.

Usage: *./wadreader path_to_wad_file [path_to_store_files]*

path_to_store_files default is *path_to_wad_file + _unpacked*

# gfx2img
Convert gfx + pal textures to png image

Usage: *./gfx2img path_to_gfx_file [--pal custom_pallete_file] [--output_file_name]*
