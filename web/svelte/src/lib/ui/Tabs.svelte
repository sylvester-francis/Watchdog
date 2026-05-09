<script lang="ts">
  interface Option {
    value: string;
    label: string;
  }

  interface Props {
    options: Option[];
    value?: string;
    onchange?: (newValue: string) => void;
  }

  let { options, value = $bindable(''), onchange }: Props = $props();

  function selectTab(newValue: string) {
    value = newValue;
    onchange?.(newValue);
  }
</script>

<div role="tablist" data-active={value} class="flex gap-1 border-b border-border">
  {#each options as opt (opt.value)}
    <button
      role="tab"
      type="button"
      aria-selected={opt.value === value}
      data-value={opt.value}
      onclick={() => selectTab(opt.value)}
      class="px-4 py-2 text-sm font-medium transition-colors border-b-2
             {opt.value === value ? 'border-accent text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}"
    >
      {opt.label}
    </button>
  {/each}
</div>
