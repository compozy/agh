import { useId, useRef } from "react";

export function useLocalRowKeys(rowCount: number) {
  const prefix = useId();
  const nextKey = useRef(0);
  const keys = useRef<string[]>([]);
  const allocate = () => `${prefix}-watch-event-${nextKey.current++}`;

  while (keys.current.length < rowCount) keys.current.push(allocate());
  if (keys.current.length > rowCount) keys.current.length = rowCount;

  return {
    keys: keys.current,
    append: () => keys.current.push(allocate()),
    remove: (index: number) => keys.current.splice(index, 1),
  };
}
