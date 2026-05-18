export { theme } from './theme';
export type { Theme } from './theme';
export { default as SafeHTML } from './SafeHTML.svelte';
export type { SafeHTMLProps } from './SafeHTML.types';
export { cn } from './lib/utils';

// ===== NEW EXPORTS =====
// Data Table (TanStack-based)
export {
  createSvelteTable,
  FlexRender,
  renderComponent,
  renderSnippet,
} from './components/data-table/index.js';

// Pagination
export { default as Pagination } from './components/pagination/pagination.svelte';
export { default as PaginationItem } from './components/pagination/pagination-item.svelte';
export { default as PaginationLink } from './components/pagination/pagination-link.svelte';
export { default as PaginationPrevious } from './components/pagination/pagination-previous.svelte';
export { default as PaginationNext } from './components/pagination/pagination-next.svelte';
export { default as PaginationEllipsis } from './components/pagination/pagination-ellipsis.svelte';
export { default as PaginationContent } from './components/pagination/pagination-content.svelte';

// Select
export { default as Select } from './components/select/select.svelte';
export { default as SelectTrigger } from './components/select/select-trigger.svelte';
export { default as SelectContent } from './components/select/select-content.svelte';
export { default as SelectItem } from './components/select/select-item.svelte';
export { default as SelectValue } from './components/select/select-value.svelte';

// Input Group
export { default as InputGroup } from './components/input-group/input-group.svelte';
export { default as InputGroupInput } from './components/input-group/input-group-input.svelte';
export { default as InputGroupText } from './components/input-group/input-group-text.svelte';
export { default as InputGroupButton } from './components/input-group/input-group-button.svelte';

// Button Group
export { default as ButtonGroup } from './components/button-group/button-group.svelte';
export { default as ButtonGroupText } from './components/button-group/button-group-text.svelte';

// Skeleton
export { default as Skeleton } from './components/skeleton/skeleton.svelte';

// Spinner
export { default as Spinner } from './components/spinner/spinner.svelte';

// Item (list item pattern)
export { default as Item } from './components/item/item.svelte';
export { default as ItemContent } from './components/item/item-content.svelte';
export { default as ItemTitle } from './components/item/item-title.svelte';
export { default as ItemDescription } from './components/item/item-description.svelte';

// Separator
export { default as Separator } from './components/separator/separator.svelte';

// Tabs
export { default as Tabs } from './components/tabs/tabs.svelte';
export { default as TabsList } from './components/tabs/tabs-list.svelte';
export { default as TabsTrigger } from './components/tabs/tabs-trigger.svelte';
export { default as TabsContent } from './components/tabs/tabs-content.svelte';

// Radio Group
export { default as RadioGroup } from './components/radio-group/radio-group.svelte';
export { default as RadioGroupItem } from './components/radio-group/radio-group-item.svelte';

// Tooltip
export { default as Tooltip } from './components/tooltip/tooltip.svelte';
export { default as TooltipContent } from './components/tooltip/tooltip-content.svelte';
export { default as TooltipTrigger } from './components/tooltip/tooltip-trigger.svelte';

// Sonner (toast notifications)
export { default as Sonner } from './components/sonner/sonner.svelte';

// Button
export { default as Button } from './components/button/button.svelte';

// Input
export { default as Input } from './components/input/input.svelte';

// Native Select
export { default as NativeSelect } from './components/native-select/native-select.svelte';
export { default as NativeSelectOption } from './components/native-select/native-select-option.svelte';

// Badge
export { default as Badge } from './components/badge/badge.svelte';

// EmptyState
export { default as EmptyState } from './components/empty-state/empty-state.svelte';

// ConfirmDialog
export { default as ConfirmDialog } from './components/confirm-dialog/confirm-dialog.svelte';
