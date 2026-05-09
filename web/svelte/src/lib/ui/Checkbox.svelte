<script lang="ts">
  interface Props {
    checked?: boolean;
    disabled?: boolean;
    error?: boolean;
    name?: string;
    id?: string;
  }

  let {
    checked = $bindable(false),
    disabled = false,
    error = false,
    name,
    id,
  }: Props = $props();
</script>

<input
  type="checkbox"
  bind:checked
  {disabled}
  {name}
  {id}
  aria-invalid={error || undefined}
  class="w-4 h-4 accent-accent rounded border bg-card-elevated transition-colors
         focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1 focus:ring-offset-background
         disabled:opacity-50 disabled:cursor-not-allowed
         {error ? 'border-destructive' : 'border-border'}"
/>
