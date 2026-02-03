<script>
  /**
   * AnimatedCounter — Smooth number animation like a slot machine
   * Usage: <AnimatedCounter value={85} duration={800} />
   */
  import { onMount } from 'svelte';
  
  let { value = 0, duration = 600, prefix = '', suffix = '' } = $props();
  
  let displayValue = $state(0);
  let prevValue = $state(0);
  let animating = $state(false);
  
  // Easing function (ease-out cubic)
  function easeOutCubic(t) {
    return 1 - Math.pow(1 - t, 3);
  }
  
  function animateTo(target) {
    if (animating) return;
    const start = displayValue;
    const diff = target - start;
    if (diff === 0) return;
    
    animating = true;
    const startTime = performance.now();
    
    function tick(now) {
      const elapsed = now - startTime;
      const progress = Math.min(elapsed / duration, 1);
      const eased = easeOutCubic(progress);
      
      displayValue = Math.round(start + diff * eased);
      
      if (progress < 1) {
        requestAnimationFrame(tick);
      } else {
        displayValue = target;
        animating = false;
      }
    }
    
    requestAnimationFrame(tick);
  }
  
  // Watch for value changes
  $effect(() => {
    if (value !== prevValue) {
      animateTo(value);
      prevValue = value;
    }
  });
  
  onMount(() => {
    // Initial animation from 0
    setTimeout(() => animateTo(value), 100);
  });
</script>

<span class="counter" class:animating>{prefix}{displayValue}{suffix}</span>

<style>
  .counter {
    font-variant-numeric: tabular-nums;
    transition: transform 0.1s ease;
  }
  .counter.animating {
    transform: scale(1.02);
  }
</style>
