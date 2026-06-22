package ele

import "github.com/CodMac/arch-lens-dep-analyer/model"

func init() {
	RegisterEnricher(model.File, &FileMetadataEnricher{})
}
