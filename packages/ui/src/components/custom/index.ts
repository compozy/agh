// Page chrome / structure
export { PageShell, type PageShellProps, type PageShellDensity } from "./page-shell";
export { Section, type SectionProps } from "./section";
export { SplitPane, SPLIT_LIST_WIDTH_DEFAULT, type SplitPaneProps } from "./split-pane";
export {
  ListGroup,
  ListGroupHeader,
  ListGroupItems,
  ListGroupRoot,
  type ListGroupHeaderProps,
  type ListGroupItemsProps,
  type ListGroupProps,
} from "./list-group";
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
} from "./data-surface";
export {
  BlockLoading,
  type BlockLoadingProps,
  type BlockLoadingSize,
  type BlockLoadingSurface,
} from "./block-loading";

// Topbar — mounted by P4 shell
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
} from "./topbar";

// Status / signal vocabulary
export { Eyebrow, type EyebrowProps } from "./eyebrow";
export {
  Pill,
  PillDot,
  PillLink,
  pillVariants,
  type PillDotProps,
  type PillLinkProps,
  type PillProps,
  type PillSize,
  type PillTone,
} from "./pill";
export {
  PillGroup,
  pillGroupSegmentVariants,
  type PillGroupItem,
  type PillGroupProps,
  type PillGroupSize,
} from "./pill-group";
export {
  ConnectionIndicator,
  STATUS_CONFIG,
  type ConnectionIndicatorDotProps,
  type ConnectionIndicatorLabelProps,
  type ConnectionIndicatorProps,
  type ConnectionStatus,
  type ConnectionVariant,
} from "./connection-indicator";
export {
  StatusCard,
  type StatusCardActionProps,
  type StatusCardBodyProps,
  type StatusCardFooterProps,
  type StatusCardHeaderProps,
  type StatusCardProps,
  type StatusCardTone,
} from "./status-card";
export { KindChip, type KindChipProps } from "./kind-chip";
export { RightRail, type RightRailMode, type RightRailProps } from "./right-rail";

// Runtime composites
export { Metric, type MetricProps, type MetricTone } from "./metric";
export { MetricGrid, type MetricGridColumns, type MetricGridProps } from "./metric-grid";
export {
  CodeBlock,
  CopyIconButton,
  type CodeBlockHighlightState,
  type CodeBlockProps,
  type CodeBlockTone,
  type CopyIconButtonProps,
} from "./code-block";
export {
  ChatMessageBubble,
  type ChatMessageAlign,
  type ChatMessageBubbleProps,
  type ChatMessageRole,
} from "./chat-message-bubble";
export {
  TOOL_CALL_STATUS_LABEL,
  ToolCallCard,
  type ToolCallCardProps,
  type ToolCallCardSectionProps,
  type ToolCallStatus,
} from "./tool-call-card";
export { ToolCallStatusIcon, type ToolCallStatusIconProps } from "./tool-call-status-icon";
export {
  WireCard,
  WireCardBody,
  WireCardFoot,
  WireCardHead,
  type WireCardProps,
} from "./wire-card";
export { TypingDots, type TypingDotsProps } from "./typing-dots";
export {
  KindIcon,
  providerKindIconRegistry,
  type KindIconProps,
  type KindIconRegistry,
  type KindIconRegistryEntry,
  type KindIconSize,
  type KindIconTone,
} from "./kind-icon";
export { Logo, type LogoProps, type LogoVariant } from "./logo";
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
} from "./catalog-card";
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
} from "./metadata-list";
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
} from "./linked-record-table";
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
} from "./command-select";
export { SearchInput, type SearchInputProps } from "./search-input";
export {
  ConfirmDialog,
  type ConfirmDialogNoteTone,
  type ConfirmDialogProps,
  type ConfirmDialogTone,
} from "./confirm-dialog";
export { UIProvider, type UIProviderProps } from "./ui-provider";

// Net-new shared-kit composites (P3)
export { LaneTabs, type LaneTabsItem, type LaneTabsProps } from "./lane-tabs";
export { Sparkline, type SparklineProps } from "./sparkline";
export { RouteState, type RouteStateMode, type RouteStateProps } from "./route-state";
export { FieldRow, type FieldRowProps } from "./field-row";
export { ContextBox, type ContextBoxEntry, type ContextBoxProps } from "./context-box";
export { JsonViewer, type JsonViewerProps } from "./json-viewer";
export { EditorFooter, type EditorFooterProps } from "./editor-footer";
export { KpiCard, type KpiCardProps } from "./kpi-card";
export {
  StatusBreakdown,
  type StatusBreakdownItem,
  type StatusBreakdownProps,
} from "./status-breakdown";
export { MetadataTile, type MetadataTileProps } from "./metadata-tile";
export { DetailHeader, type DetailHeaderCrumb, type DetailHeaderProps } from "./detail-header";
export { FormSection, type FormSectionProps, type FormSectionSize } from "./form-section";
export { MonoId, type MonoIdProps, type MonoIdSize } from "./mono-id";
export { Time, type TimeMode, type TimeProps } from "./time";
export {
  StatusDot,
  type StatusDotProps,
  type StatusDotSize,
  type StatusDotTone,
  type StatusDotVariant,
} from "./status-dot";
export { RadioCard, type RadioCardProps } from "./radio-card";
export {
  ActionResultBanner,
  type ActionResultBannerProps,
  type ActionResultBannerTone,
} from "./action-result-banner";
export {
  StackedProgress,
  type StackedProgressProps,
  type StackedProgressSegment,
} from "./stacked-progress";
export { ReviewRow, type ReviewRowProps } from "./review-row";
export { Timeline, type TimelineProps } from "./timeline";
export { TimelineEvent, type TimelineEventProps } from "./timeline-event";
export { PriorityBars, type PriorityBarsProps, type PriorityLevel } from "./priority-bars";
export {
  OperationalLinksRow,
  type OperationalLink,
  type OperationalLinksRowProps,
} from "./operational-links-row";

// Content primitives — markdown + run/tool/avatar surfaces.
export { Markdown, STREAMDOWN_SAFE_CONFIG, type MarkdownProps } from "./markdown";
export { DescriptionCard, type DescriptionCardProps } from "./description-card";
export {
  RUN_STATUS_LABEL,
  RUN_STATUS_TONE,
  RunCard,
  type RunCardProps,
  type RunCardStatus,
  type RunCardWarning,
  type RunCardWarningTone,
} from "./run-card";
export { OwnerAvatar, type OwnerAvatarProps, type OwnerAvatarSize } from "./owner-avatar";

// Slot + system primitives.
export { RestartBanner, type RestartBannerProps, type RestartBannerTone } from "./restart-banner";
export { PageActionsTopbarSlot, type PageActionsTopbarSlotProps } from "./page-actions-topbar-slot";
export {
  StatusLineTopbarSlot,
  type StatusLineTopbarSlotItem,
  type StatusLineTopbarSlotProps,
} from "./status-line-topbar-slot";
export {
  DETAIL_INSPECTOR_INLINE_BREAKPOINT,
  DETAIL_INSPECTOR_INLINE_WIDTH,
  DetailInspector,
  type DetailInspectorProps,
  type DetailInspectorTab,
} from "./detail-inspector";
export {
  QueueHealthSparkline,
  type QueueHealthSparklineBucket,
  type QueueHealthSparklineProps,
} from "./queue-health-sparkline";
