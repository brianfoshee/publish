package imgur

/* ***Galleries and Photos***
--pics-path is similar to --blog-path, it's the dir where photo gallery md and photo md and jpg files are

New flag, --preparePics=path, which renames each picture with a new shortID and creates a md file with that name too.
The md file will have some basic info like the gallery it belongs to (can get this from the base dir) and the ID, with
empty fields that need to be filled in.

On build, go through pics path folders to find a base gallery folder containing gallery md files and image/md files.
Parse gallery md and generate gallery pages. For every image in the gallery folder, add it to the gallery data structure.
For every image, run retrobatch via command line to convert original image into new files in the build directory folder.
  * get from http://flyingmeat.com/download/latest/#retrobatch
Validate: a md file with the name of the folder exists (08/iceland-year-off/ must contain iceland-year-off.md)
*/

func Build(path string, drafts bool) {
}
