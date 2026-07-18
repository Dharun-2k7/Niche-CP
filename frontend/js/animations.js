// =========================================================================
// THE QUANTUM ARENA - Minimal Scroll & Viewport Animations
// =========================================================================

document.addEventListener('DOMContentLoaded', () => {
    // Only initialize if GSAP is loaded
    if (typeof gsap === 'undefined' || typeof ScrollTrigger === 'undefined') {
        console.warn("GSAP libraries not loaded.");
        return;
    }

    gsap.registerPlugin(ScrollTrigger);

    // =========================================================================
    // 1. HERO ENTRANCE
    // =========================================================================
    const heroTitle = document.querySelector(".hero-split");
    if (heroTitle) {
        gsap.from(".hero-split", {
            opacity: 0,
            y: 30,
            scale: 0.98,
            duration: 1.0,
            ease: "power3.out"
        });
    }

    if (document.querySelector('.hero-subtitle')) {
        const tl = gsap.timeline();
        tl.from(".hero-subtitle", {
            opacity: 0,
            y: 15,
            duration: 0.5,
            ease: "power2.out"
        }, "-=0.4")
        .from(".hero-cta .btn-magnetic", {
            opacity: 0,
            y: 10,
            stagger: 0.1,
            duration: 0.4,
            ease: "power2.out"
        }, "-=0.3");
    }

    // =========================================================================
    // 2. VIEWPORT FADE-INS (Cards & Sections)
    // =========================================================================
    // 1. Staggered Grid Reveals (General)
    gsap.utils.toArray(".bento-grid:not(#transform-grid), .stats-grid, .values-grid").forEach(grid => {
        gsap.from(grid.children, {
            scrollTrigger: {
                trigger: grid,
                start: "top 85%",
                toggleActions: "play none none reverse"
            },
            y: 20,
            scale: 0.98,
            opacity: 0,
            duration: 0.5,
            stagger: 0.1,
            ease: "power2.out"
        });
    });

    // 1b. Transformation Engine (Left-to-Right Progression)
    const transformGrid = document.getElementById('transform-grid');
    if (transformGrid) {
        gsap.from(transformGrid.children, {
            scrollTrigger: {
                trigger: transformGrid,
                start: "top 80%",
                toggleActions: "play none none reverse"
            },
            x: -40,
            opacity: 0,
            duration: 0.6,
            stagger: 0.15,
            ease: "power2.out"
        });
    }

    // 1c. Why NicheCP Custom Storytelling
    const comparisonPairs = gsap.utils.toArray(".comparison-grid .comparison-pair");
    if (comparisonPairs.length >= 4) {
        // Pair 1: Bottom Up
        gsap.from(comparisonPairs[0], {
            scrollTrigger: { trigger: comparisonPairs[0], start: "top 85%", toggleActions: "play none none reverse" },
            y: 50, opacity: 0, duration: 0.6, ease: "power2.out"
        });
        // Pair 2: Slide from Left
        gsap.from(comparisonPairs[1], {
            scrollTrigger: { trigger: comparisonPairs[1], start: "top 85%", toggleActions: "play none none reverse" },
            x: -50, opacity: 0, duration: 0.6, ease: "power2.out"
        });
        // Pair 3: Slide from Right
        gsap.from(comparisonPairs[2], {
            scrollTrigger: { trigger: comparisonPairs[2], start: "top 85%", toggleActions: "play none none reverse" },
            x: 50, opacity: 0, duration: 0.6, ease: "power2.out"
        });
        // Pair 4: Bloom Scale
        gsap.from(comparisonPairs[3], {
            scrollTrigger: { trigger: comparisonPairs[3], start: "top 85%", toggleActions: "play none none reverse" },
            scale: 0.9, opacity: 0, duration: 0.6, ease: "power2.out"
        });
    }

    // 2. Individual Text/Header Reveals
    gsap.utils.toArray(".section-container h2, .section-container > p, .vision-content p").forEach(el => {
        gsap.from(el, {
            scrollTrigger: {
                trigger: el,
                start: "top 90%",
                toggleActions: "play none none reverse"
            },
            y: 20,
            opacity: 0,
            duration: 0.6,
            ease: "power2.out"
        });
    });

    // =========================================================================
    // 3. NUMBER COUNTERS
    // =========================================================================
    const counters = document.querySelectorAll('.counter');
    counters.forEach(counter => {
        const targetValue = parseInt(counter.getAttribute('data-target'), 10) || 0;
        const obj = { val: 0 };
        gsap.to(obj, {
            val: targetValue,
            duration: 1.5,
            ease: "power2.out",
            scrollTrigger: {
                trigger: counter,
                start: "top 85%"
            },
            onUpdate: function() {
                counter.innerHTML = Math.floor(obj.val);
            }
        });
    });

    // =========================================================================
    // 4. VISION SECTION TEXT FADE
    // =========================================================================
    gsap.utils.toArray('.cinematic-text').forEach(text => {
        gsap.to(text, {
            color: "rgba(255,255,255,1)",
            scrollTrigger: {
                trigger: text,
                start: "top 85%",
                end: "top 50%",
                scrub: true
            }
        });
    });
});
