<svelte:options runes={true} />

<script lang="ts">
  import type { ComponentProps } from 'svelte';

  import InlineForm from '@ui/components/form/InlineForm/InlineForm.svelte';

  type Props = Partial<ComponentProps<typeof InlineForm>>;

  const {
    inputProps = undefined,
    submitLabel = 'Submit',
    submitButtonProps = undefined,
    trailingAction = undefined,
    className = undefined,
    value: _value = '',
  }: Props = $props();

  let value = $state('');
  let submittedValue = $state('');

  const handleValueChange = (nextValue: string) => {
    value = nextValue;
  };

  const handleSubmitValue = (nextValue: string) => {
    submittedValue = nextValue;
    value = '';
  };
</script>

<div>
  <InlineForm
    {className}
    {value}
    {inputProps}
    {submitLabel}
    {submitButtonProps}
    {trailingAction}
    onValueChange={handleValueChange}
    onSubmitValue={handleSubmitValue}
  />
  <p style="margin-top: 0.75rem; color: #666;">
    Last submitted: {submittedValue === '' ? 'None' : submittedValue}
  </p>
</div>
