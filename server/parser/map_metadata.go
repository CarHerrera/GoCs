package parser

import (
	"fmt"

	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
)

// extraMapMetadata covers maps ex.GetMapMetadata doesn't bundle - it only
// ships the official competitive pool, but this project's overview images
// (client/public/overviews) cover many workshop/community maps and CS2
// reworks (e.g. de_cache_new.bsp) on top of that. ex.GetMapMetadata panics
// for anything it doesn't have, so these are sourced independently (from
// the map's own HLTV overview file, pos_x/pos_y/scale) and used as a
// fallback instead.
//
// de_cache: https://github.com/MurkyYT/cs2-map-icons/blob/main/data/radar_info/de_cache.txt
var extraMapMetadata = map[string]ex.Map{
	"de_cache": {PosX: -2000, PosY: 3250, Scale: 5.5},
}

// ResolveMapMetadata resolves radar-overview calibration for name, falling
// back to extraMapMetadata for maps ex.GetMapMetadata doesn't bundle, and
// returning an error instead of letting the library's panic reach the
// caller if neither source has it.
func ResolveMapMetadata(name string) (meta ex.Map, err error) {
	defer func() {
		if r := recover(); r != nil {
			if fallback, ok := extraMapMetadata[name]; ok {
				meta, err = fallback, nil
				return
			}
			err = fmt.Errorf("no radar metadata available for map %q: %v", name, r)
		}
	}()
	return ex.GetMapMetadata(name), nil
}
