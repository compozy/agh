import { createContext, use } from "react";
import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import type { ItemInstance, TreeInstance } from "@headless-tree/core";

import { cn } from "@agh/ui/lib/utils";
import { MinusIcon, PlusIcon, ChevronDownIcon } from "lucide-react";

type ToggleIconType = "chevron" | "plus-minus";

// TreeInstance and ItemInstance are invariant in T (they expose write paths
// like updateCachedData that take T), so the shared context type erases T to
// `any`. Each consumer hook re-narrows on read via a generic cast.
interface TreeContextValue {
  indent: number;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  currentItem?: ItemInstance<any>;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  tree?: TreeInstance<any>;
  toggleIconType: ToggleIconType;
}

const TreeContext = createContext<TreeContextValue>({
  indent: 20,
  currentItem: undefined,
  tree: undefined,
  toggleIconType: "plus-minus",
});

interface TypedTreeContext<T> {
  indent: number;
  currentItem?: ItemInstance<T>;
  tree?: TreeInstance<T>;
  toggleIconType: ToggleIconType;
}

function useTreeContext<T>(): TypedTreeContext<T> {
  return use(TreeContext) as TypedTreeContext<T>;
}

function optionalFeatureCall<T, K extends keyof ItemInstance<T>>(
  item: ItemInstance<T>,
  method: K
): boolean | undefined {
  const candidate = (item as unknown as Record<string, unknown>)[method as string];
  if (typeof candidate !== "function") return undefined;
  return Boolean((candidate as () => boolean).call(item));
}

interface TreeProps<T> extends Omit<React.HTMLAttributes<HTMLDivElement>, "children"> {
  indent?: number;
  tree: TreeInstance<T>;
  toggleIconType?: ToggleIconType;
  children?: React.ReactNode;
}

function Tree<T>({
  indent = 20,
  tree,
  className,
  toggleIconType = "chevron",
  ...props
}: TreeProps<T>) {
  const containerProps = tree.getContainerProps();
  const { style: mergedContainerStyle, ...otherProps } = mergeProps<"div">(containerProps, props);

  const mergedStyle = {
    ...mergedContainerStyle,
    "--tree-indent": `${indent}px`,
  } as React.CSSProperties;

  return (
    <TreeContext.Provider value={{ indent, tree, toggleIconType }}>
      <div
        data-slot="tree"
        style={mergedStyle}
        className={cn("flex flex-col", className)}
        {...otherProps}
      />
    </TreeContext.Provider>
  );
}

interface TreeItemProps<T> extends Omit<useRender.ComponentProps<"button">, "indent"> {
  item: ItemInstance<T>;
  indent?: number;
}

function TreeItem<T>({ item, className, render, children, ...props }: TreeItemProps<T>) {
  const parentContext = useTreeContext<T>();
  const { indent } = parentContext;

  const itemProps = item.getProps();
  const { style: mergedItemStyle, ...otherProps } = mergeProps<"button">(itemProps, {
    ...props,
    children,
  });

  const mergedStyle = {
    ...mergedItemStyle,
    "--tree-padding": `${item.getItemMeta().level * indent}px`,
  } as React.CSSProperties;

  // Feature methods (drag/search/selection) only exist when the matching
  // feature is registered with useTree. Guard the optional features so trees
  // without them (e.g. selection-only) don't crash at render time.
  const isFolder = item.isFolder();
  const focused = optionalFeatureCall(item, "isFocused") ?? false;
  const selected = optionalFeatureCall(item, "isSelected") ?? false;
  const dragTarget = optionalFeatureCall(item, "isDragTarget");
  const searchMatch = optionalFeatureCall(item, "isMatchingSearch");
  const defaultProps = {
    "data-slot": "tree-item",
    type: "button" as const,
    style: mergedStyle,
    className: cn(
      "z-10 ps-(--tree-padding) outline-hidden select-none not-last:pb-0.5 focus:z-20 data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
      className
    ),
    "data-focus": focused,
    "data-folder": isFolder,
    "data-selected": selected,
    "data-drag-target": dragTarget,
    "data-search-match": searchMatch,
    "aria-expanded": isFolder ? item.isExpanded() : undefined,
  };

  return (
    <TreeContext.Provider value={{ ...parentContext, currentItem: item }}>
      {useRender({
        defaultTagName: "button",
        render,
        props: mergeProps<"button">(defaultProps, otherProps),
      })}
    </TreeContext.Provider>
  );
}

interface TreeItemLabelProps<T> extends React.HTMLAttributes<HTMLSpanElement> {
  item?: ItemInstance<T>;
}

function TreeItemLabel<T>({
  item: propItem,
  children,
  className,
  ...props
}: TreeItemLabelProps<T>) {
  const { currentItem, toggleIconType } = useTreeContext<T>();
  const item = propItem ?? currentItem;

  if (!item) return null;

  const isFolder = item.isFolder();
  const isExpanded = item.isExpanded();

  return (
    <span
      data-slot="tree-item-label"
      className={cn(
        "in-focus-visible:ring-ring/50 bg-background hover:bg-accent in-data-[selected=true]:bg-accent in-data-[selected=true]:text-accent-foreground in-data-[drag-target=true]:bg-accent flex items-center gap-1 transition-colors not-in-data-[folder=true]:ps-7 in-focus-visible:ring-[3px] in-data-[search-match=true]:bg-blue-50! [&_svg]:pointer-events-none [&_svg]:shrink-0",
        "rounded-sm",
        "py-1.5",
        "px-2",
        "text-sm",
        className
      )}
      {...props}
    >
      {isFolder &&
        (toggleIconType === "plus-minus" ? (
          isExpanded ? (
            <MinusIcon
              className="text-muted-foreground size-3.5"
              stroke="currentColor"
              strokeWidth="1"
            />
          ) : (
            <PlusIcon
              className="text-muted-foreground size-3.5"
              stroke="currentColor"
              strokeWidth="1"
            />
          )
        ) : (
          <ChevronDownIcon className="text-muted-foreground size-4 in-aria-[expanded=false]:-rotate-90" />
        ))}
      {children ?? item.getItemName()}
    </span>
  );
}

interface TreeDragLineProps<T> extends React.HTMLAttributes<HTMLDivElement> {
  tree?: TreeInstance<T>;
}

function TreeDragLine<T>({ className, tree: propTree, ...props }: TreeDragLineProps<T>) {
  const context = useTreeContext<T>();
  const tree = propTree ?? context.tree;

  if (!tree || typeof (tree as { getDragLineStyle?: unknown }).getDragLineStyle !== "function") {
    return null;
  }

  const dragLine = (
    tree as TreeInstance<T> & {
      getDragLineStyle: (topOffset?: number, leftOffset?: number) => React.CSSProperties;
    }
  ).getDragLineStyle();
  return (
    <div
      style={dragLine}
      className={cn(
        "bg-primary before:bg-background before:border-primary absolute z-30 -mt-px h-0.5 w-[unset] before:absolute before:-top-[3px] before:left-0 before:size-2 before:border-2",
        "before:rounded-full",
        className
      )}
      {...props}
    />
  );
}

export { Tree, TreeItem, TreeItemLabel, TreeDragLine };
export type { TreeProps, TreeItemProps, TreeItemLabelProps, TreeDragLineProps };
