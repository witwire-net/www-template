<svelte:options runes={true} />

<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  import Button from '@ui/components/atoms/Button/Button.svelte';

  import styles from './FileUpload.module.scss';

  type Props = Omit<HTMLInputAttributes, 'accept' | 'class' | 'multiple' | 'onchange' | 'type'> & {
    label?: string;
    helperText?: string;
    error?: string;
    multiple?: boolean;
    accept?: string;
    title?: string;
    subtitle?: string;
    dragActiveTitle?: string;
    dragActiveSubtitle?: string;
    buttonLabel?: string;
    enableDrop?: boolean;
    onFilesChange?: (files: File[], source: 'drop' | 'input') => void;
    class?: string;
    className?: string;
  };

  const joinClassName = (...values: (string | undefined)[]) =>
    values.filter((value) => value !== undefined && value !== '').join(' ');

  const {
    label = undefined,
    helperText = undefined,
    error = undefined,
    multiple = false,
    accept = undefined,
    title = undefined,
    subtitle = undefined,
    dragActiveTitle = undefined,
    dragActiveSubtitle = undefined,
    buttonLabel = 'Choose files',
    enableDrop = true,
    onFilesChange = undefined,
    class: classProp = undefined,
    className = undefined,
    ...restProps
  }: Props = $props();

  let inputElement: HTMLInputElement | null = null;
  let files = $state<File[]>([]);
  let isDragActive = $state(false);

  const hasLabel = $derived(label !== undefined && label !== '');
  const hasError = $derived(error !== undefined && error !== '');
  const hasHelperText = $derived(helperText !== undefined && helperText !== '');
  const rootClassName = $derived(joinClassName(styles.wrapper, classProp, className));
  const dropzoneClassName = $derived(
    joinClassName(
      styles.dropzone,
      hasError ? styles.error : undefined,
      isDragActive ? styles.dropzoneActive : undefined
    )
  );
  const resolvedTitle = $derived(
    isDragActive
      ? dragActiveTitle ?? 'Release to upload files'
      : title ?? (enableDrop ? 'Drop files here or browse from your device' : 'Select files to upload')
  );
  const resolvedSubtitle = $derived(
    isDragActive
      ? dragActiveSubtitle ?? 'Dropped files will be added to the current selection.'
      : subtitle ??
          (enableDrop
            ? multiple
              ? 'Drag files in or use the button to select multiple files.'
              : 'Drag a file in or use the button to select one from your device.'
            : multiple
              ? 'Use the button below to select multiple files.'
              : 'Use the button below to select a file.')
  );

  const normalizeFiles = (nextFiles: readonly File[]): File[] => {
    return multiple ? [...nextFiles] : nextFiles.slice(0, 1);
  };

  const syncInputFiles = (nextFiles: readonly File[]) => {
    if (inputElement === null || typeof DataTransfer === 'undefined') {
      return;
    }

    const dataTransfer = new DataTransfer();

    for (const file of nextFiles) {
      dataTransfer.items.add(file);
    }

    inputElement.files = dataTransfer.files;
  };

  const updateFiles = (nextFiles: readonly File[], source: 'drop' | 'input') => {
    const normalizedFiles = normalizeFiles(nextFiles);

    files = [...normalizedFiles];
    syncInputFiles(normalizedFiles);
    onFilesChange?.([...normalizedFiles], source);
  };

  const handleButtonClick = () => {
    inputElement?.click();
  };

  const handleChange = (event: Event) => {
    const target = event.currentTarget;
    if (!(target instanceof HTMLInputElement)) {
      updateFiles([], 'input');
      return;
    }

    updateFiles(target.files === null ? [] : Array.from(target.files), 'input');
  };

  const handleDragOver = (event: DragEvent) => {
    if (!enableDrop) {
      return;
    }

    event.preventDefault();
    isDragActive = true;
  };

  const handleDragLeave = () => {
    if (!enableDrop) {
      return;
    }

    isDragActive = false;
  };

  const handleDrop = (event: DragEvent) => {
    if (!enableDrop) {
      return;
    }

    event.preventDefault();
    isDragActive = false;

    const droppedFiles = event.dataTransfer?.files;
    updateFiles(droppedFiles === undefined ? [] : Array.from(droppedFiles), 'drop');
  };
</script>

<div class={rootClassName}>
  {#if hasLabel}
    <span class={styles.label ?? ''}>{label}</span>
  {/if}
  <div
    class={dropzoneClassName}
    role="group"
    aria-label={resolvedTitle}
    ondragover={handleDragOver}
    ondragleave={handleDragLeave}
    ondrop={handleDrop}
  >
    <div class={styles.content ?? ''}>
      <div class={styles.title ?? ''}>{resolvedTitle}</div>
      <div class={styles.subtitle ?? ''}>{resolvedSubtitle}</div>
      <Button type="button" size="sm" variant="secondary" onclick={handleButtonClick}>
        {buttonLabel}
      </Button>
    </div>
    <input
      bind:this={inputElement}
      type="file"
      class={styles.input ?? ''}
      {multiple}
      {accept}
      onchange={handleChange}
      {...restProps}
    />
  </div>
  {#if files.length > 0}
    <ul class={styles.fileList ?? ''}>
      {#each files as file (`${file.name}-${file.lastModified}-${file.size}`)}
        <li class={styles.fileItem ?? ''}>{file.name}</li>
      {/each}
    </ul>
  {/if}
  {#if hasError}
    <span class={styles.errorMessage ?? ''}>{error}</span>
  {:else if hasHelperText}
    <span class={styles.helperText ?? ''}>{helperText}</span>
  {/if}
</div>
