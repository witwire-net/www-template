<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import CommandPalette from './CommandPalette.svelte';

  let open = $state(false);
  let lastCommand = $state('No command yet');
</script>

<div style="padding: 1.5rem;">
  <Button
    onclick={() => {
      open = true;
    }}
  >
    Open Command Palette
  </Button>

  <p style="margin-top: 1rem; color: #666;">Last command: {lastCommand}</p>

  <CommandPalette
    {open}
    onClose={() => {
      open = false;
    }}
    inputPlaceholder="Type a command"
    commands={[
      {
        label: 'Create project',
        description: 'Start a new workspace project',
        onSelect: () => {
          lastCommand = 'Create project';
        },
      },
      {
        label: 'Invite member',
        description: 'Add teammates',
        onSelect: () => {
          lastCommand = 'Invite member';
        },
      },
      {
        label: 'Open settings',
        description: 'Configure workspace',
        onSelect: () => {
          lastCommand = 'Open settings';
        },
      },
    ]}
  />
</div>
