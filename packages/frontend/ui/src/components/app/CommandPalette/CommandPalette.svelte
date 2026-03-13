<svelte:options runes={true} />

<script lang="ts">
  import styles from './CommandPalette.module.scss';

  type CommandItem = {
    description?: string;
    label: string;
    onSelect?: () => void;
  };

  type Props = {
    commands: CommandItem[];
    inputPlaceholder?: string;
    onClose: () => void;
    open: boolean;
  };

  let {
    open,
    onClose,
    commands,
    inputPlaceholder = undefined,
  }: Props = $props();

  let query = $state('');
  let inputElement = $state<HTMLInputElement | null>(null);

  const filteredCommands = $derived(
    commands.filter((command) => command.label.toLowerCase().includes(query.toLowerCase()))
  );

  function closePalette(): void {
    onClose();
  }

  function handleBackdropKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter' || event.key === ' ' || event.key === 'Escape') {
      event.preventDefault();
      closePalette();
    }
  }

  function handleInput(event: Event & { currentTarget: EventTarget & HTMLInputElement }): void {
    query = event.currentTarget.value;
  }

  function handleSelect(command: CommandItem): void {
    command.onSelect?.();
    closePalette();
  }

  $effect(() => {
    if (typeof document === 'undefined' || !open) {
      return;
    }

    const handleKeydown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        closePalette();
      }
    };

    document.addEventListener('keydown', handleKeydown);

    return () => {
      document.removeEventListener('keydown', handleKeydown);
    };
  });

  $effect(() => {
    if (!open || inputElement === null) {
      return;
    }

    inputElement.focus();
  });
</script>

{#if open}
  <div class={styles.overlay ?? ''}>
    <button
      type="button"
      class={styles.backdrop ?? ''}
      aria-label="Close command palette"
      onclick={closePalette}
      onkeydown={handleBackdropKeydown}
    ></button>
    <div class={styles.palette ?? ''}>
      <input
        bind:this={inputElement}
        type="search"
        class={styles.input ?? ''}
        placeholder={inputPlaceholder}
        value={query}
        oninput={handleInput}
      />
      <div class={styles.list ?? ''}>
        {#each filteredCommands as command (command.label)}
          <button type="button" class={styles.item ?? ''} onclick={() => handleSelect(command)}>
            <div class={styles.label ?? ''}>{command.label}</div>
            {#if command.description !== undefined && command.description !== ''}
              <div class={styles.description ?? ''}>{command.description}</div>
            {/if}
          </button>
        {/each}
      </div>
    </div>
  </div>
{/if}
