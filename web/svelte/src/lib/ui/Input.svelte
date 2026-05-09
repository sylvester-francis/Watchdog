<script lang="ts">
  interface Props {
    value?: string | number;
    type?: 'text' | 'email' | 'password' | 'url' | 'number' | 'search';
    placeholder?: string;
    disabled?: boolean;
    readonly?: boolean;
    error?: boolean;
    name?: string;
    id?: string;
    min?: number | string;
    max?: number | string;
    step?: number | string;
  }

  let {
    value = $bindable(''),
    type = 'text',
    placeholder,
    disabled = false,
    readonly = false,
    error = false,
    name,
    id,
    min,
    max,
    step,
  }: Props = $props();
</script>

<input
  bind:value
  {type}
  {placeholder}
  {disabled}
  {readonly}
  {name}
  {id}
  {min}
  {max}
  {step}
  aria-invalid={error || undefined}
  class="w-full bg-card-elevated border rounded px-3 py-2 text-foreground placeholder:text-muted-foreground transition-colors
         focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1 focus:ring-offset-background
         disabled:opacity-50 disabled:cursor-not-allowed
         {error ? 'border-destructive' : 'border-border'}"
/>
