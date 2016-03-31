### Current status: Alpha

- Archives
  - *.pak
    - [x] Extract files ([UnPacker](#unpacker))
    - [ ] Pack files
  - *_WAD
    - [x] Extract files ([WadReader](#wadreader))
    - [ ] Pack files
- Models
  - [ ] Extraction
    - [ ] Vertex data
    - [ ] Textures data
    - [ ] Physics
    - [ ] Animation
    - [ ] Meta
- Textures 
  - [x] Coverting TXR_(GFX+PAL) textures to *.png images (with lods) ([gfx2img](#gfx2img))
  - [ ] Converting *.png to _GFX+_PAL

# UnPacker
Tool for unpaking part\*.pak files using info from *GODOFWAR.TOC*

Autodetecting version of GoW (GoW1 or GoW2)

Usage: *./god_of_war_tools.exe unpack -in ../GOW_DIR_WITH_TOK_FILE/*

Help: *./god_of_war_tools.exe unpack -h*

Formats in archive:

| Format | Info |
|-------:|:-----|
| PSS/PSW | mpeg videos (without sound). PSS without headers (or not)|
| WAD | game archives, can use [Wadreader](#Wadreader) to unpack |
| VAG/VA1-5 | VAGp ADPCM sounds (depended on language) |
| VPK | RAW ADPCM music |
| TXT | SANITY.TXT used for check archive validity |

After unpaking, summary size of all files being lower then size of archive. This is because, archive dublicate files for faster access on disk. (use -l option for see how much files is duplicated)

# WadReader
Tool for extracting files from *.wad archives. At this moment not extract all information.

Autodetecting version of GoW (GoW1 or GoW2)

Usage: *./god_of_war_tools.exe extract -wad ../ARCHIVE.WAD*

Help: *./god_of_war_tools.exe extract -h*

# gfx2img
Convert gfx + pal textures to png image

Both game (GoW1 + GoW2) use same textures format

Usage: *./god_of_war_tools.exe image -txr ../TXR_texture*

Help: *./god_of_war_tools.exe image -h*
