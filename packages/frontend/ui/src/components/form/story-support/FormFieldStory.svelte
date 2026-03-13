<svelte:options runes={true} />

<script lang="ts">
  import Input from '@ui/components/atoms/Input/Input.svelte';
  import FormField from '@ui/components/form/FormField/FormField.svelte';

  type Props = {
    label?: string;
    helperText?: string;
    error?: string;
    required?: boolean;
    htmlFor?: string;
  };

  const {
    label = undefined,
    helperText = undefined,
    error = undefined,
    required = false,
    htmlFor = 'form-field-story-input',
  }: Props = $props();
</script>

<FormField {label} {helperText} {error} {required} {htmlFor}>
  <Input id={htmlFor} placeholder="name@company.com" />
</FormField>
