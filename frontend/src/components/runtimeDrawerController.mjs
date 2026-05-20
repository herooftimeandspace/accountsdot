import { useCallback, useState } from "react";

export function runtimeDrawerItemId(item, getId = (value) => value?.id) {
  const id = getId(item);
  return id === undefined || id === null ? "" : String(id);
}

export function nextRuntimeDrawerSelection(currentItem, nextItem, getId) {
  if (!nextItem) {
    return null;
  }
  return runtimeDrawerItemId(currentItem, getId) === runtimeDrawerItemId(nextItem, getId) ? null : nextItem;
}

export function nextRuntimeDrawerSelectionForId(selectedId, nextItem, getId) {
  if (!nextItem) {
    return null;
  }
  return String(selectedId ?? "") === runtimeDrawerItemId(nextItem, getId) ? null : nextItem;
}

export function useRuntimeDrawerSelection(initialItem = null, getId) {
  const [selectedItem, setSelectedItem] = useState(initialItem);
  const selectItem = useCallback((nextItem) => {
    setSelectedItem((currentItem) => nextRuntimeDrawerSelection(currentItem, nextItem, getId));
  }, [getId]);
  const closeDrawer = useCallback(() => setSelectedItem(null), []);
  return { selectedItem, setSelectedItem, selectItem, closeDrawer };
}
