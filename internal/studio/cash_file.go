package studio

import (
	"bytes"
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
)

const cashBodySize = 16 + 5*cash.Slots

// Preserve wire strings that the edit did not touch, including legacy
// encodings and NUL padding. Edit the fixed fields with GX's canonical writer.
func encodeCash(original []byte, products []cash.Product) ([]byte, error) {
	encoded, err := sdata.EncodeCash(products)
	if err != nil {
		return nil, err
	}
	previous, err := sdata.ParseCash(original)
	if err != nil {
		return nil, err
	}
	old := layoutReader{raw: original, pos: 4}
	type rawProduct struct {
		product cash.Product
		fields  [3][]byte
	}
	byCode := make(map[string]rawProduct, len(previous))
	ambiguous := make(map[string]bool)
	for _, product := range previous {
		row := rawProduct{product: product}
		old.take(cashBodySize)
		for i := range row.fields {
			start := old.pos
			old.text()
			row.fields[i] = old.raw[start:old.pos]
		}
		if _, exists := byCode[product.Code]; exists {
			ambiguous[product.Code] = true
		}
		byCode[product.Code] = row
	}
	if old.err != nil {
		return nil, old.err
	}
	next := layoutReader{raw: encoded, pos: 4}
	result := bytes.Clone(encoded[:4])
	for _, product := range products {
		result = append(result, next.take(cashBodySize)...)
		before, found := byCode[product.Code]
		beforeValues := [3]string{before.product.Name, before.product.Code, before.product.Description}
		newValues := [3]string{product.Name, product.Code, product.Description}
		for i := range newValues {
			start := next.pos
			next.text()
			if !found || ambiguous[product.Code] {
				result = append(result, encoded[start:next.pos]...)
			} else if newValues[i] == beforeValues[i] {
				result = append(result, before.fields[i]...)
			} else {
				wire := before.fields[i]
				_, err := encodeField(Field{}, newValues[i], nil, &wire, nil)
				if err != nil {
					return nil, fmt.Errorf("product %q: %w", product.Code, err)
				}
				result = append(result, wire...)
			}
		}
	}
	return result, next.err
}

func validateClientItems(project *Project, products []cash.Product) error {
	path, err := FindFile(project.DataDir, "Item/Item.SData")
	if err != nil {
		return err
	}
	raw, err := readSData(path)
	if err != nil {
		return err
	}
	items, err := sdata.ParseItems(raw)
	if err != nil {
		return fmt.Errorf("Item.SData: %w", err)
	}
	for _, product := range products {
		for i, item := range product.Items {
			if item.ItemID == 0 {
				continue
			}
			key := sdata.ItemKey{Type: uint8(item.ItemID / 1000), TypeID: uint8(item.ItemID % 1000)}
			if items[key] == nil {
				return fmt.Errorf("product %q slot %d: item %d is missing in Item.SData", product.Code, i+1, item.ItemID)
			}
		}
	}
	return nil
}
