<script lang="ts">
  interface Props {
    value?: string;
    rows?: number;
    placeholder?: string;
    disabled?: boolean;
    error?: boolean;
    name?: string;
    id?: string;
  }

  let {
    value = $bindable(''),
    rows = 3,
    placeholder,
    disabled = false,
    error = false,
    name,
    id,
  }: Props = $props();
</script>

<textarea
  bind:value
  {rows}
  {placeholder}
  {disabled}
  {name}
  {id}
  aria-invalid={error || undefined}
  class="w-full bg-card-elevated border rounded px-3 py-2 text-foreground placeholder:text-muted-foreground transition-colors
         focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1 focus:ring-offset-background
         disabled:opacity-50 disabled:cursor-not-allowed
         {error ? 'border-destructive' : 'border-border'}"
></textarea>
