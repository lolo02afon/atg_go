package channel_duplicate

import (
	"reflect"
	"strings"

	"github.com/gotd/td/tg"
)

// cloneEntities создаёт глубокую копию среза сущностей сообщения.
func cloneEntities(src []tg.MessageEntityClass) []tg.MessageEntityClass {
	if len(src) == 0 {
		return nil
	}
	dst := make([]tg.MessageEntityClass, len(src))
	for i, e := range src {
		v := reflect.New(reflect.TypeOf(e).Elem())
		v.Elem().Set(reflect.ValueOf(e).Elem())
		dst[i] = v.Interface().(tg.MessageEntityClass)
	}
	return dst
}

// adjustEntitiesAfterRemoval сдвигает смещения сущностей после удаления фрагмента текста.
// Предполагается, что удаляемый фрагмент не содержит собственного форматирования.
func adjustEntitiesAfterRemoval(entities []tg.MessageEntityClass, original, remove string) []tg.MessageEntityClass {
	if remove == "" || len(entities) == 0 {
		return entities
	}
	remUTF := utf16Len(remove)
	totalShift := 0
	searchFrom := 0
	res := entities
	for {
		idx := strings.Index(original[searchFrom:], remove)
		if idx == -1 {
			break
		}
		start := searchFrom + idx
		offsetUTF := utf16Len(original[:start]) - totalShift
		tmp := make([]tg.MessageEntityClass, 0, len(res))
		for _, ent := range res {
			off, length := getOffsetLength(ent)
			switch {
			case off >= offsetUTF+remUTF:
				setOffsetLength(ent, off-remUTF, length)
				tmp = append(tmp, ent)
			case off+length <= offsetUTF:
				tmp = append(tmp, ent)
				// сущность пересекает удалённый фрагмент — пропускаем
			}
		}
		res = tmp
		searchFrom = start + len(remove)
		totalShift += remUTF
	}
	return res
}

// getOffsetLength возвращает смещение и длину сущности.
func getOffsetLength(ent tg.MessageEntityClass) (int, int) {
	v := reflect.ValueOf(ent).Elem()
	return int(v.FieldByName("Offset").Int()), int(v.FieldByName("Length").Int())
}

// setOffsetLength задаёт смещение и длину сущности.
func setOffsetLength(ent tg.MessageEntityClass, off, length int) {
	v := reflect.ValueOf(ent).Elem()
	v.FieldByName("Offset").SetInt(int64(off))
	v.FieldByName("Length").SetInt(int64(length))
}
