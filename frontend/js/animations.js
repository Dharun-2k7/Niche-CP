// =========================================================================
// THE QUANTUM ARENA - ScrollTrigger Masterclass
// =========================================================================

document.addEventListener('DOMContentLoaded', () => {
    // Only initialize if GSAP is loaded
    if (typeof gsap === 'undefined' || typeof ScrollTrigger === 'undefined') {
        console.warn("GSAP libraries not loaded.");
        return;
    }

    gsap.registerPlugin(ScrollTrigger);

    // =========================================================================
    // 1. HERO ENTRANCE (Custom Text Split & Parallax)
    // =========================================================================
    
    // Custom SplitText implementation (since GSAP SplitText is premium)
    const heroTitle = document.querySelector(".hero-split");
    if (heroTitle) {
        const text = heroTitle.innerHTML;
        // Split by words, preserving br and spans
        // For simplicity, we will just fade up the whole block, or split line by line
        // Actually, let's just fade up the whole title smoothly to avoid messing up HTML tags
        gsap.from(".hero-split", {
            opacity: 0,
            y: 50,
            rotateX: -45,
            duration: 1.2,
            ease: "back.out(1.5)",
            transformOrigin: "50% 100%"
        });
    }

    gsap.from(".hero-subtitle", {
        opacity: 0,
        y: 20,
        duration: 0.8,
        ease: "power3.out"
    }, "-=0.4")
    .from(".hero-cta .btn-magnetic", {
        opacity: 0,
        scale: 0.9,
        stagger: 0.1,
        duration: 0.6,
        ease: "back.out(1.5)"
    }, "-=0.6");

    // Floating Nodes Parallax based on mouse movement
    const pNodes = document.querySelectorAll('.p-node');
    if (window.innerWidth > 768 && pNodes.length > 0) {
        window.addEventListener('mousemove', (e) => {
            const x = (e.clientX / window.innerWidth - 0.5) * 2;
            const y = (e.clientY / window.innerHeight - 0.5) * 2;
            
            gsap.to('.node-1', { x: x * 60, y: y * 60, duration: 2, ease: 'power2.out' });
            gsap.to('.node-2', { x: x * -40, y: y * -40, duration: 2, ease: 'power2.out' });
            gsap.to('.node-3', { x: x * 80, y: y * 80, duration: 2, ease: 'power2.out', rotation: x * 15 });
        });
    }

    // =========================================================================
    // 2. BENTO GRID ENTRANCE (Scale & Stagger)
    // =========================================================================
    gsap.from(".bento-card", {
        scrollTrigger: {
            trigger: ".bento-grid",
            start: "top 80%",
            toggleActions: "play none none reverse"
        },
        y: 100,
        opacity: 0,
        scale: 0.9,
        duration: 0.8,
        stagger: {
            amount: 0.3,
            from: "random"
        },
        ease: "power3.out"
    });

    // Heatmap staggered lighting
    gsap.from(".heatmap-node.active", {
        scrollTrigger: {
            trigger: ".heatmap",
            start: "top 90%",
            toggleActions: "play none none reverse"
        },
        backgroundColor: "rgba(255,255,255,0.05)",
        boxShadow: "0 0 0px transparent",
        duration: 0.1,
        stagger: 0.1,
        ease: "none"
    });

    // =========================================================================
    // 3. HORIZONTAL SCROLL STORYTELLING
    // =========================================================================
    const timelineContainer = document.querySelector(".timeline-container");
    if (timelineContainer && window.innerWidth > 768) {
        gsap.to(timelineContainer, {
            x: () => -(timelineContainer.scrollWidth - window.innerWidth) + "px",
            ease: "none",
            scrollTrigger: {
                trigger: ".timeline-wrapper",
                pin: true,
                scrub: 1,
                end: () => "+=" + (timelineContainer.scrollWidth - window.innerWidth),
                invalidateOnRefresh: true
            }
        });
    }

    // =========================================================================
    // 4. ECOSYSTEM EXPANSION ANIMATIONS
    // =========================================================================

    // A. Number Counters (Why CP Matters)
    const counters = document.querySelectorAll('.counter');
    counters.forEach(counter => {
        gsap.to(counter, {
            innerHTML: counter.getAttribute('data-target'),
            snap: { innerHTML: 1 },
            duration: 2,
            ease: "power2.out",
            scrollTrigger: {
                trigger: counter,
                start: "top 85%"
            }
        });
    });

    // B. SVG Journey Line Drawing
    const lineFill = document.querySelector('.journey-line-fill');
    if (lineFill) {
        gsap.set(lineFill, { strokeDasharray: 800, strokeDashoffset: 800 });
        gsap.to(lineFill, {
            strokeDashoffset: 0,
            ease: "none",
            scrollTrigger: {
                trigger: ".journey-map",
                start: "top 60%",
                end: "bottom 80%",
                scrub: 1
            }
        });
    }

    // C. Journey Phase Reveals
    gsap.utils.toArray('.journey-phase').forEach(phase => {
        const isRight = phase.classList.contains('right');
        gsap.from(phase, {
            x: isRight ? 100 : -100,
            opacity: 0,
            duration: 1,
            ease: "back.out(1.5)",
            scrollTrigger: {
                trigger: phase,
                start: "top 85%"
            }
        });
    });

    // D. Sticky Transformation Engine (8 Stages)
    if (window.innerWidth > 768) {
        const slides = gsap.utils.toArray('.success-slide');
        const evolutionFill = document.getElementById('evolutionFill');
        const evolutionLabel = document.getElementById('evolutionLabel');
        
        if (slides.length > 0) {
            ScrollTrigger.create({
                trigger: ".success-timeline-wrapper",
                start: "top top",
                end: "+=700%",
                pin: ".success-sticky-container",
                scrub: true,
                onUpdate: (self) => {
                    const progress = self.progress;
                    const currentStage = Math.min(Math.floor(progress * slides.length) + 1, slides.length);
                    if (evolutionFill) evolutionFill.style.width = (currentStage / slides.length * 100) + '%';
                    if (evolutionLabel) evolutionLabel.textContent = `Stage ${currentStage} / ${slides.length}`;
                }
            });
            
            const step = 100 / slides.length;
            slides.forEach((slide, i) => {
                const tl = gsap.timeline({
                    scrollTrigger: {
                        trigger: ".success-timeline-wrapper",
                        start: `top+=${i * step}% top`,
                        end: `top+=${(i + 1) * step}% top`,
                        scrub: true
                    }
                });
                
                if (i === slides.length - 1) {
                    tl.to(slide, { opacity: 1, y: 0, duration: 0.5 });
                } else {
                    tl.to(slide, { opacity: 1, y: 0, duration: 0.5 })
                      .to(slide, { opacity: 0, y: -100, duration: 0.5 });
                }
            });
        }
    }

    // E. Vision Section Cinematic Fade
    gsap.utils.toArray('.cinematic-text').forEach(text => {
        gsap.to(text, {
            color: "rgba(255,255,255,1)",
            scrollTrigger: {
                trigger: text,
                start: "top 80%",
                end: "top 40%",
                scrub: true
            }
        });
    });

    // F. Values Bento Grid
    gsap.from(".values-grid .bento-card", {
        scrollTrigger: {
            trigger: ".values-grid",
            start: "top 85%"
        },
        y: 100,
        opacity: 0,
        duration: 0.8,
        stagger: 0.2,
        ease: "power3.out"
    });

});
