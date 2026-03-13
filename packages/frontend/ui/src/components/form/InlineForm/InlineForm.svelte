<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes, HTMLFormAttributes, HTMLInputAttributes } from 'svelte/elements';

  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Input from '@ui/components/atoms/Input/Input.svelte';

  import styles from './InlineForm.module.scss';

  const noop = (): void => {
    return;
  };

  type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
  type ButtonSize = 'sm' | 'md' | 'lg';
  type Renderable = Snippet | string | number | null | undefined;

  type InlineFormInputProps = Omit<HTMLInputAttributes, 'class' | 'oninput' | 'size' | 'value'> & {
    className?: string;
    error?: string;
    fullWidth?: boolean;
    helperText?: string;
    label?: string;
    size?: 'sm' | 'md' | 'lg';
  };

  type InlineFormButtonProps = Omit<HTMLButtonAttributes, 'class' | 'type'> & {
    children?: Renderable;
    className?: string;
    fullWidth?: boolean;
    isLoading?: boolean;
    size?: ButtonSize;
    variant?: ButtonVariant;
  };

  type Props = Omit<HTMLFormAttributes, 'class' | 'onsubmit'> & {
    value: string;
    onValueChange?: (value: string, event: Event) => void;
    onSubmitValue?: (value: string, event: SubmitEvent) => void;
    onSubmit?: (event: SubmitEvent) => void;
    inputProps?: InlineFormInputProps;
    inputLabel?: string;
    inputPlaceholder?: string;
    inputType?: HTMLInputElement['type'];
    submitLabel?: string;
    submitContent?: Renderable;
    submitVariant?: ButtonVariant;
    submitSize?: ButtonSize;
    submitButtonProps?: InlineFormButtonProps;
    trailingAction?: string;
    trailing?: Snippet;
    className?: string;
    inputContainerClassName?: string;
    inputClassName?: string;
    actionClassName?: string;
    trailingClassName?: string;
  };

  const joinClassName = (...values: (string | undefined)[]) =>
    values.filter((value) => value !== undefined && value !== '').join(' ');

  const hasRenderable = (value: Renderable): boolean => {
    return value !== undefined && value !== null;
  };

  const isSnippet = (value: Renderable): value is Snippet => {
    return typeof value === 'function';
  };

  const getTextContent = (value: Renderable): string => {
    if (typeof value === 'string') {
      return value;
    }

    if (typeof value === 'number') {
      return String(value);
    }

    return '';
  };

  const {
    value,
    onValueChange = noop,
    onSubmitValue = undefined,
    onSubmit = undefined,
    inputProps = undefined,
    inputLabel = undefined,
    inputPlaceholder = undefined,
    inputType = 'text',
    submitLabel = 'Submit',
    submitContent = undefined,
    submitVariant = 'primary',
    submitSize = 'sm',
    submitButtonProps = undefined,
    trailingAction = undefined,
    trailing,
    className = undefined,
    inputContainerClassName = undefined,
    inputClassName = undefined,
    actionClassName = undefined,
    trailingClassName = undefined,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(joinClassName(styles.form, className));
  const resolvedInputClassName = $derived(joinClassName(inputClassName, inputProps?.className));
  const resolvedButtonContent = $derived(submitContent ?? submitButtonProps?.children ?? submitLabel);
  const submitButtonClassName = $derived(submitButtonProps?.className);
  const resolvedSubmitButtonProps = $derived.by(() => {
    const {
      children: _submitButtonChildren = undefined,
      className: _submitButtonClassName = undefined,
      size: _submitButtonSize = undefined,
      variant: _submitButtonVariant = undefined,
      ...submitButtonRestProps
    }: InlineFormButtonProps = submitButtonProps ?? {};

    return submitButtonRestProps;
  });
  const hasTrailingAction = $derived(
    trailing !== undefined || (trailingAction !== undefined && trailingAction !== '')
  );

  const handleInput = (event: Event) => {
    const target = event.currentTarget;
    if (target instanceof HTMLInputElement) {
      onValueChange(target.value, event);
    }
  };

  const handleSubmit = (event: SubmitEvent) => {
    onSubmit?.(event);
    if (event.defaultPrevented) {
      return;
    }

    event.preventDefault();
    onSubmitValue?.(value, event);
  };
</script>

<form {...restProps} class={rootClassName} onsubmit={handleSubmit}>
  <div class={joinClassName(styles.input, inputContainerClassName)}>
    <Input
      {...inputProps}
      type={inputProps?.type ?? inputType}
      label={inputProps?.label ?? inputLabel}
      placeholder={inputProps?.placeholder ?? inputPlaceholder}
      value={value}
      className={resolvedInputClassName}
      oninput={handleInput}
    />
  </div>
  <div class={joinClassName(styles.action, actionClassName)}>
    <Button
      {...resolvedSubmitButtonProps}
      type="submit"
      variant={submitButtonProps?.variant ?? submitVariant}
      size={submitButtonProps?.size ?? submitSize}
      className={submitButtonClassName}
    >
      {#if hasRenderable(resolvedButtonContent)}
        {#if isSnippet(resolvedButtonContent)}
          {@render resolvedButtonContent()}
        {:else}
          {getTextContent(resolvedButtonContent)}
        {/if}
      {/if}
    </Button>
  </div>
  {#if hasTrailingAction}
    <div class={joinClassName(styles.trailing, trailingClassName)}>
      {#if trailing !== undefined}
        {@render trailing()}
      {:else}
        {trailingAction}
      {/if}
    </div>
  {/if}
</form>
