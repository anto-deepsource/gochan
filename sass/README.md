## Theme development using Sass
If you want, you can install [Sass](https://sass-lang.com/install) to streamline writing CSS stylesheets. It requires node.js as a dependency so if you don't want to install it and Sass's dependencies (and its dependencies' dependencies,...) the CSS files generated by Sass are provided.

To use sass, run `./build.py sass`. If you want to minify the created css files, use the `--minify` flag. If you want sass to watch the input directory for changes as you edit and save the files, use the `--watch` flag.

If you are upgading from gochan 2.2, delete your html/css directory unless you have made themes that you want to keep. Then rebuild the pages. (/manage?action=rebuildall)

## Attribution
The BunkerChan, Clear, and Dark themes come from the imageboard BunkerChan. Burichan is based on the theme with the same name from Kusaba X. Photon comes from nol.ch (I think?) that as far as I know no longer exists. Pipes was created by a user (Mbyte?) on Lunachan (defunct). Yotsuba and Yotsuba B are based on the themes with the same names from 4chan.