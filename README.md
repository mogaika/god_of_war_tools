### Current status: Alpha

- Archives
  - *.pak
    - [x] Unpack files ([UnPacker](#unpacker))
    - [ ] Pack files
  - *_WAD
    - [x] Extract files ([WadReader](#wadreader))
		- Models
			- [x] Vertexes
			- [x] Normals
			- [x] Textures
			- [ ] Physics
			- [ ] Animation
			- [ ] Joints
			- [ ] Shadowbox
		- Materials
			- [x] Texture image *.png
			- [ ] Material information
			- [ ] Animation
		- Scripts
		- Sounds
		- [x] Videos

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
| TXT | SANITY.TXT used to check data |

After unpaking, summary size of all files being lower then size of archive. This is because, archive dublicate files for faster access on disk. (use -l option for see how much files is duplicated)

# Extractor
Tool for extracting files from *.wad archives.
Convert files to known file types with tree saving:
- PNG
- OBJ
- MTL

If argument *-dump true* presented, dump all files.

Autodetecting version of GoW (GoW1 or GoW2)
At this moment primary supports only GoW1

Usage: *./god_of_war_tools.exe extract -wad ../ARCHIVE.WAD -out ./outDirectory -dump* 

Help: *./god_of_war_tools.exe extract -h*
