---
title: Incus Compose 
layout: hextra-home
---

{{< hextra/hero-badge >}}
  <div class="hx-w-2 hx-h-2 hx-rounded-full hx-bg-primary-400"></div>
  <span>Free, open source</span>
  {{< icon name="arrow-circle-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}

<div class="hx-mt-6 hx-mb-6">
{{< hextra/hero-headline >}}
  Track your deployed containers&nbsp;<br class="sm:hx-block hx-hidden" />and virtual machines.
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
  Lightweight service daemon&nbsp;<br class="sm:hx-block hx-hidden" />with a modern dashboard
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mb-6">
{{< hextra/hero-button text="Get Started" link="docs" >}}
</div>

<div class="hx-mt-6"></div>

{{< hextra/feature-grid cols=2 >}}
  {{< hextra/feature-card
    title="Dashboard for your Home Lab"
    subtitle="See all your deployments on a single page or look at each server individually."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
    image="images/dashboard.png"
    imageClass="hx-top-[40%] hx-left-[24px] hx-w-[180%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(194,97,254,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Terminal Friendly"
    subtitle="Search your deployments using the terminal you already have open."
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-lg:hx-min-h-[340px]"
    image="images/terminal.png"
    imageClass="hx-top-[40%] hx-left-[36px] hx-w-[180%] sm:hx-w-[110%] dark:hx-opacity-80"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(142,53,74,0.15),hsla(0,0%,100%,0));"
  >}}

  {{< hextra/feature-card
    title="No Dependencies"
    subtitle="Inventory ships as a static binary. Download, extract, and put in your PATH, no complex installation required."
  >}}
  {{< hextra/feature-card
    title="Supports Docker and Incus"
    subtitle="Automatically track your Docker and Incus deployments. More platforms coming soon..."
  >}}

{{< /hextra/feature-grid >}}
