import { useId, useRef } from "react";

export function useLocalRowKeys(rows: readonly unknown[], label: string) {
  const prefix = useId();
  const nextKey = useRef(0);
  const keys = useRef<string[]>([]);
  const allocate = () => `${prefix}-${label}-${nextKey.current++}`;

  while (keys.current.length < rows.length) keys.current.push(allocate());
  if (keys.current.length > rows.length) keys.current.length = rows.length;

  return {
    keys: keys.current,
    append: () => keys.current.push(allocate()),
    remove: (index: number) => keys.current.splice(index, 1),
  };
}
