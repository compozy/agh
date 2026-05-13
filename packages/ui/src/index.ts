// Utility
export { cn } from "./lib/utils";

// Components
export { Button, buttonVariants } from "./components/button";
export {
  Card,
  CardHeader,
  CardFooter,
  CardTitle,
  CardAction,
  CardDescription,
  CardContent,
} from "./components/card";
export type { CardProps, CardSize } from "./components/card";
export { Input } from "./components/input";
export { Label } from "./components/label";
export { Separator, type SeparatorProps } from "./components/separator";
export { Skeleton, SkeletonRows, type SkeletonRowsProps } from "./components/skeleton";
export { Spinner } from "./components/spinner";
export {
  Alert,
  AlertTitle,
  AlertDescription,
  AlertAction,
  AlertActions,
  AlertMeta,
  alertVariants,
  type AlertProps,
} from "./components/alert";
export {
  Progress,
  ProgressTrack,
  ProgressIndicator,
  ProgressLabel,
  ProgressValue,
} from "./components/progress";
export {
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
} from "./components/table";
export { Kbd, KbdGroup } from "./components/kbd";
export { UIProvider, type UIProviderProps } from "./components/custom/ui-provider";
export { Logo, type LogoProps, type LogoVariant } from "./components/custom/logo";
export {
  KindIcon,
  bridgeKindIconRegistry,
  providerKindIconRegistry,
  type KindIconProps,
  type KindIconRegistry,
  type KindIconRegistryEntry,
  type KindIconSize,
  type KindIconTone,
} from "./components/custom/kind-icon";
export {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
  type DialogChromeVariant,
  type DialogContentProps,
  type DialogFooterProps,
  type DialogHeaderProps,
} from "./components/dialog";
export {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "./components/popover";
export {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "./components/sheet";
export { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./components/tooltip";
export { Tabs, TabsContent, TabsList, TabsTrigger, type TabsTriggerProps } from "./components/tabs";
export { ScrollArea, ScrollBar } from "./components/scroll-area";
export {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectScrollDownButton,
  SelectScrollUpButton,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "./components/select";
export {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxClear,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxLabel,
  ComboboxList,
  ComboboxSeparator,
  ComboboxTrigger,
  ComboboxValue,
  useComboboxAnchor,
} from "./components/combobox";
export {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut,
} from "./components/command";
export {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "./components/dropdown-menu";
export { Switch } from "./components/switch";
export { Toggle, toggleVariants } from "./components/toggle";
export { ToggleGroup, ToggleGroupItem } from "./components/toggle-group";
export {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "./components/accordion";
export { Collapsible, CollapsibleContent, CollapsibleTrigger } from "./components/collapsible";
export {
  Sidebar,
  SidebarSectionLabel,
  SIDEBAR_COLLAPSE_BREAKPOINT_DEFAULT,
  SIDEBAR_PANEL_WIDTH_DEFAULT,
  SIDEBAR_PANEL_WIDTH_MD,
  SIDEBAR_PANEL_WIDTH_MD_BREAKPOINT,
  SIDEBAR_RAIL_WIDTH,
  useSidebarViewport,
  type SidebarProps,
  type SidebarViewport,
} from "./components/sidebar";
export {
  SplitPane,
  SPLIT_LIST_WIDTH_DEFAULT,
  type SplitPaneProps,
} from "./components/custom/split-pane";
export { PageShell, type PageShellProps } from "./components/custom/page-shell";
export { Eyebrow, type EyebrowProps } from "./components/custom/eyebrow";
export {
  Pill,
  PillDot,
  PillLink,
  pillVariants,
  type PillProps,
  type PillDotProps,
  type PillLinkProps,
  type PillTone,
  type PillSize,
} from "./components/custom/pill";
export {
  PillGroup,
  pillGroupSegmentVariants,
  type PillGroupProps,
  type PillGroupItem,
  type PillGroupSize,
} from "./components/custom/pill-group";
export { SearchInput, type SearchInputProps } from "./components/custom/search-input";
export { Empty, type EmptyProps } from "./components/empty";
export { Section, type SectionProps } from "./components/custom/section";

// Topbar — dormant code in P3; mounted by P4 shell.
export {
  Topbar,
  TopbarOverflowIcon,
  TopbarSlotContext,
  TopbarSlotProvider,
  useTopbarSlot,
  useTopbarSlotContext,
  useTopbarSlotValue,
  type TopbarProps,
  type TopbarRouteContext,
  type TopbarSlotContextValue,
  type TopbarSlotProviderProps,
  type TopbarSlotValue,
} from "./components/custom/topbar";

// Promoted from `web/src/systems/network/components/`.
export { KindChip, type KindChipProps } from "./components/custom/kind-chip";
export { RightRail, type RightRailMode, type RightRailProps } from "./components/custom/right-rail";

// Net-new shared-kit composites (P3).
export { LaneTabs, type LaneTabsItem, type LaneTabsProps } from "./components/custom/lane-tabs";
export { Sparkline, type SparklineProps } from "./components/custom/sparkline";
export {
  RouteState,
  type RouteStateMode,
  type RouteStateProps,
} from "./components/custom/route-state";
export { FieldRow, type FieldRowProps } from "./components/custom/field-row";
export {
  ContextBox,
  type ContextBoxEntry,
  type ContextBoxProps,
} from "./components/custom/context-box";
export { JsonViewer, type JsonViewerProps } from "./components/custom/json-viewer";
export { EditorFooter, type EditorFooterProps } from "./components/custom/editor-footer";
export { KpiCard, type KpiCardProps } from "./components/custom/kpi-card";
export {
  StatusBreakdown,
  type StatusBreakdownItem,
  type StatusBreakdownProps,
} from "./components/custom/status-breakdown";
export { MetadataTile, type MetadataTileProps } from "./components/custom/metadata-tile";
export {
  DetailHeader,
  type DetailHeaderCrumb,
  type DetailHeaderProps,
} from "./components/custom/detail-header";
export {
  FormSection,
  type FormSectionProps,
  type FormSectionSize,
} from "./components/custom/form-section";
export { Icon, type IconProps, type IconSize } from "./components/icon";
export { MonoId, type MonoIdProps, type MonoIdSize } from "./components/custom/mono-id";
export { Time, type TimeMode, type TimeProps } from "./components/custom/time";
export {
  StatusDot,
  type StatusDotProps,
  type StatusDotSize,
  type StatusDotTone,
  type StatusDotVariant,
} from "./components/custom/status-dot";
export {
  formatAbsoluteTime,
  formatDuration,
  formatRelativeTime,
  FORMAT_TIME_FALLBACK,
} from "./lib/format-time";
export {
  AGH_CODE_DEFAULT_THEME,
  AGH_CODE_SUPPORTED_LANGUAGES,
  AGH_CODE_THEMES,
  normalizeAghCodeLanguage,
  resolveAghCodeThemeName,
  type AghCodeLanguage,
  type AghCodeThemeName,
  type CodeBlockResolvedTheme,
  type CodeBlockThemeMode,
} from "./lib/code-theme";
export {
  AGENT_SLOT_COUNT,
  HUMAN_SLOT_COUNT,
  SYSTEM_SLOT_COUNT,
  colorsFor,
  seed,
  type OwnerColors,
  type OwnerKind,
} from "./lib/owner-palette";
export {
  DescriptionCard,
  STREAMDOWN_SAFE_CONFIG,
  type DescriptionCardProps,
} from "./components/custom/description-card";
export { StreamMarkdown, type StreamMarkdownProps } from "./components/custom/stream-markdown";
export {
  RUN_STATUS_LABEL,
  RUN_STATUS_TONE,
  RunCard,
  type RunCardProps,
  type RunCardStatus,
  type RunCardWarning,
  type RunCardWarningTone,
} from "./components/custom/run-card";
export {
  OwnerAvatar,
  type OwnerAvatarProps,
  type OwnerAvatarSize,
} from "./components/custom/owner-avatar";
export {
  RestartBanner,
  type RestartBannerProps,
  type RestartBannerTone,
} from "./components/custom/restart-banner";
export {
  PageActionsTopbarSlot,
  type PageActionsTopbarSlotProps,
} from "./components/custom/page-actions-topbar-slot";
export {
  StatusLineTopbarSlot,
  type StatusLineTopbarSlotItem,
  type StatusLineTopbarSlotProps,
} from "./components/custom/status-line-topbar-slot";
export {
  DETAIL_INSPECTOR_INLINE_BREAKPOINT,
  DETAIL_INSPECTOR_INLINE_WIDTH,
  DetailInspector,
  type DetailInspectorProps,
  type DetailInspectorTab,
} from "./components/custom/detail-inspector";
export {
  QueueHealthSparkline,
  type QueueHealthSparklineBucket,
  type QueueHealthSparklineProps,
} from "./components/custom/queue-health-sparkline";
export { RadioCard, type RadioCardProps } from "./components/custom/radio-card";
export {
  ActionResultBanner,
  type ActionResultBannerProps,
  type ActionResultBannerTone,
} from "./components/custom/action-result-banner";
export {
  StackedProgress,
  type StackedProgressProps,
  type StackedProgressSegment,
} from "./components/custom/stacked-progress";
export { ReviewRow, type ReviewRowProps } from "./components/custom/review-row";
export { Timeline, type TimelineProps } from "./components/custom/timeline";
export { TimelineEvent, type TimelineEventProps } from "./components/custom/timeline-event";
export {
  PriorityBars,
  type PriorityBarsProps,
  type PriorityLevel,
} from "./components/custom/priority-bars";
export {
  OperationalLinksRow,
  type OperationalLink,
  type OperationalLinksRowProps,
} from "./components/custom/operational-links-row";
export {
  WireCard,
  WireCardHead,
  WireCardBody,
  WireCardFoot,
  type WireCardProps,
} from "./components/custom/wire-card";
export { TypingDots, type TypingDotsProps } from "./components/custom/typing-dots";
export {
  CodeBlock,
  CopyIconButton,
  type CodeBlockHighlightState,
  type CodeBlockProps,
  type CodeBlockTone,
  type CopyIconButtonProps,
} from "./components/custom/code-block";
export {
  BlockLoading,
  type BlockLoadingProps,
  type BlockLoadingSize,
  type BlockLoadingSurface,
} from "./components/custom/block-loading";
export {
  DataSurface,
  DataSurfaceContent,
  DataSurfaceEmpty,
  DataSurfaceError,
  DataSurfaceLoading,
  resolveDataSurfaceState,
  type DataSurfaceContentProps,
  type DataSurfaceEmptyProps,
  type DataSurfaceErrorProps,
  type DataSurfaceLoadingProps,
  type DataSurfaceProps,
  type DataSurfaceState,
} from "./components/custom/data-surface";
export {
  ConnectionIndicator,
  type ConnectionIndicatorDotProps,
  type ConnectionIndicatorLabelProps,
  type ConnectionIndicatorProps,
  type ConnectionStatus,
  type ConnectionVariant,
} from "./components/custom/connection-indicator";
export {
  StatusCard,
  type StatusCardActionProps,
  type StatusCardBodyProps,
  type StatusCardFooterProps,
  type StatusCardHeaderProps,
  type StatusCardProps,
  type StatusCardTone,
} from "./components/custom/status-card";
export {
  ConfirmDialog,
  type ConfirmDialogNoteTone,
  type ConfirmDialogProps,
  type ConfirmDialogTone,
} from "./components/custom/confirm-dialog";
export {
  CatalogCard,
  type CatalogCardActionsProps,
  type CatalogCardDescriptionProps,
  type CatalogCardLogoProps,
  type CatalogCardLogoSize,
  type CatalogCardMetaProps,
  type CatalogCardProps,
  type CatalogCardTitleProps,
  type CatalogCardTone,
} from "./components/custom/catalog-card";
export {
  ListGroup,
  ListGroupHeader,
  ListGroupItems,
  ListGroupRoot,
  type ListGroupHeaderProps,
  type ListGroupItemsProps,
  type ListGroupProps,
} from "./components/custom/list-group";
export {
  CommandSelect,
  CommandSelectChip,
  CommandSelectChipStrip,
  CommandSelectGroup,
  CommandSelectShell,
  CommandSelectTrigger,
  type CommandSelectChipProps,
  type CommandSelectChipStripProps,
  type CommandSelectGroupProps,
  type CommandSelectProps,
  type CommandSelectShellProps,
  type CommandSelectTriggerProps,
} from "./components/custom/command-select";
export {
  MetadataList,
  MetadataListRoot,
  MetadataListRow,
  MetadataListTerm,
  MetadataListValue,
  type MetadataListProps,
  type MetadataListRowProps,
  type MetadataListTermProps,
  type MetadataListValueProps,
} from "./components/custom/metadata-list";
export {
  LinkedRecordTable,
  LinkedRecordTableBody,
  LinkedRecordTableCell,
  LinkedRecordTableOpenCell,
  LinkedRecordTableRoot,
  LinkedRecordTableRow,
  LinkedRecordTableTitle,
  type LinkedRecordTableBodyProps,
  type LinkedRecordTableCellProps,
  type LinkedRecordTableOpenCellProps,
  type LinkedRecordTableProps,
  type LinkedRecordTableRowProps,
  type LinkedRecordTableTitleProps,
} from "./components/custom/linked-record-table";
export {
  ChatMessageBubble,
  type ChatMessageBubbleProps,
  type ChatMessageRole,
  type ChatMessageAlign,
} from "./components/custom/chat-message-bubble";
export {
  TOOL_CALL_STATUS_LABEL,
  TOOL_CALL_STATUS_TONE,
  ToolCallCard,
  type ToolCallCardProps,
  type ToolCallCardSectionProps,
  type ToolCallStatus,
} from "./components/custom/tool-call-card";
export { Metric, type MetricProps, type MetricTone } from "./components/custom/metric";
export {
  MetricGrid,
  type MetricGridColumns,
  type MetricGridProps,
} from "./components/custom/metric-grid";
export {
  Avatar,
  AvatarBadge,
  AvatarFallback,
  AvatarGroup,
  AvatarGroupCount,
  AvatarImage,
  type AvatarShape,
  type AvatarSize,
} from "./components/avatar";
export {
  Breadcrumb,
  BreadcrumbEllipsis,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "./components/breadcrumb";
export {
  ButtonGroup,
  ButtonGroupSeparator,
  ButtonGroupText,
  buttonGroupVariants,
} from "./components/button-group";
export {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSeparator,
  FieldSet,
  FieldTitle,
} from "./components/field";
export {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  InputGroupText,
  InputGroupTextarea,
} from "./components/input-group";
export {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemFooter,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemSelectionIndicator,
  ItemTitle,
  type ItemAs,
  type ItemIndicator,
  type ItemIndicatorTone,
  type ItemProps,
  type ItemSelectionIndicatorProps,
} from "./components/item";
export { NativeSelect, NativeSelectOptGroup, NativeSelectOption } from "./components/native-select";
export { Tree, TreeItem, TreeItemLabel, TreeDragLine } from "./components/reui/tree";
export type {
  TreeProps,
  TreeItemProps,
  TreeItemLabelProps,
  TreeDragLineProps,
} from "./components/reui/tree";
export { Textarea, type TextareaProps, type TextareaVariant } from "./components/textarea";
export { Toaster, toast, type ToasterProps } from "./components/sonner";
export { DirectionProvider, useDirection } from "./components/direction";
