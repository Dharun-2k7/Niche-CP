// =========================================================================
// NicheCP - Three.js Particle Hero Visualization
// =========================================================================

document.addEventListener('DOMContentLoaded', () => {
    const container = document.getElementById('three-canvas-container');
    if (!container) return;
    if (typeof THREE === 'undefined') {
        console.warn("Three.js not loaded.");
        return;
    }

    // Determine particle count based on screen size
    let PARTICLE_COUNT = 35000;
    if (window.innerWidth < 1024) PARTICLE_COUNT = 18000;
    if (window.innerWidth < 768) PARTICLE_COUNT = 8000;

    const scene = new THREE.Scene();
    
    // Set up camera
    const camera = new THREE.PerspectiveCamera(60, window.innerWidth / window.innerHeight, 0.1, 1000);
    camera.position.z = 50;

    // Set up renderer
    const renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true, powerPreference: "high-performance" });
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    container.appendChild(renderer.domElement);

    // =========================================================================
    // BACKGROUND SPHERE (Procedural look via basic shaders)
    // =========================================================================
    const sphereGeometry = new THREE.SphereGeometry(25, 64, 64);
    
    // A simple custom shader material to mimic a Fresnel rim lighting effect
    const vertexShader = `
        varying vec3 vNormal;
        varying vec3 vPositionNormal;
        void main() {
            vNormal = normalize(normalMatrix * normal);
            vPositionNormal = normalize((modelViewMatrix * vec4(position, 1.0)).xyz);
            gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
        }
    `;

    const fragmentShader = `
        varying vec3 vNormal;
        varying vec3 vPositionNormal;
        void main() {
            float intensity = pow(0.65 - dot(vNormal, vec3(0, 0, 1.0)), 4.0);
            vec3 glowColor = vec3(0.1, 0.3, 0.8) * intensity;
            gl_FragColor = vec4(glowColor, intensity * 0.5);
        }
    `;

    const sphereMaterial = new THREE.ShaderMaterial({
        vertexShader: vertexShader,
        fragmentShader: fragmentShader,
        blending: THREE.AdditiveBlending,
        transparent: true,
        depthWrite: false
    });

    const backgroundSphere = new THREE.Mesh(sphereGeometry, sphereMaterial);
    scene.add(backgroundSphere);

    // =========================================================================
    // PARTICLE SYSTEM (Swarm / Sphere / Wave)
    // =========================================================================
    const geometry = new THREE.BufferGeometry();
    const positions = new Float32Array(PARTICLE_COUNT * 3);
    
    // Target positions for morphing
    const targetSwarm = new Float32Array(PARTICLE_COUNT * 3);
    const targetSphere = new Float32Array(PARTICLE_COUNT * 3);
    const targetWave = new Float32Array(PARTICLE_COUNT * 3);
    
    // Initialize positions
    for (let i = 0; i < PARTICLE_COUNT; i++) {
        const i3 = i * 3;
        
        // Random Swarm
        targetSwarm[i3] = (Math.random() - 0.5) * 100;
        targetSwarm[i3 + 1] = (Math.random() - 0.5) * 100;
        targetSwarm[i3 + 2] = (Math.random() - 0.5) * 100;

        // Sphere
        const phi = Math.acos(-1 + (2 * i) / PARTICLE_COUNT);
        const theta = Math.sqrt(PARTICLE_COUNT * Math.PI) * phi;
        const r = 28; // Slightly larger than background sphere
        targetSphere[i3] = r * Math.cos(theta) * Math.sin(phi);
        targetSphere[i3 + 1] = r * Math.sin(theta) * Math.sin(phi);
        targetSphere[i3 + 2] = r * Math.cos(phi);

        // Wave
        const x = (Math.random() - 0.5) * 80;
        const z = (Math.random() - 0.5) * 80;
        const y = Math.sin(x * 0.1) * Math.cos(z * 0.1) * 10;
        targetWave[i3] = x;
        targetWave[i3 + 1] = y;
        targetWave[i3 + 2] = z;

        // Start at Swarm
        positions[i3] = targetSwarm[i3];
        positions[i3 + 1] = targetSwarm[i3 + 1];
        positions[i3 + 2] = targetSwarm[i3 + 2];
    }

    geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    
    // Add Morph Targets
    geometry.morphAttributes.position = [];
    geometry.morphAttributes.position[0] = new THREE.BufferAttribute(targetSphere, 3);
    geometry.morphAttributes.position[1] = new THREE.BufferAttribute(targetWave, 3);
    geometry.morphAttributes.position[2] = new THREE.BufferAttribute(targetSwarm, 3);

    const material = new THREE.PointsMaterial({
        size: 0.15,
        color: 0x3b82f6,
        transparent: true,
        opacity: 0.8,
        blending: THREE.AdditiveBlending,
        depthWrite: false
    });

    const particles = new THREE.Points(geometry, material);
    scene.add(particles);

    // =========================================================================
    // MORPHING LOGIC
    // =========================================================================
    let currentTarget = 0;
    particles.morphTargetInfluences[0] = 0;
    particles.morphTargetInfluences[1] = 0;
    particles.morphTargetInfluences[2] = 0;

    function morphTo(targetIndex) {
        if (typeof gsap === 'undefined') return;
        
        // Reset all influences smoothly
        for (let i = 0; i < 3; i++) {
            if (i !== targetIndex) {
                gsap.to(particles.morphTargetInfluences, {
                    [i]: 0,
                    duration: 3,
                    ease: "power2.inOut"
                });
            }
        }
        
        // Morph to target
        gsap.to(particles.morphTargetInfluences, {
            [targetIndex]: 1,
            duration: 3,
            ease: "power2.inOut"
        });
        
        currentTarget = targetIndex;
    }

    // Start morph cycle
    setTimeout(() => morphTo(0), 1000); // Morph to sphere initially
    setInterval(() => {
        const nextTarget = (currentTarget + 1) % 3;
        morphTo(nextTarget);
    }, 15000); // 15 seconds

    // =========================================================================
    // MOUSE INTERACTION & ANIMATION
    // =========================================================================
    let mouseX = 0;
    let mouseY = 0;
    let targetX = 0;
    let targetY = 0;
    const windowHalfX = window.innerWidth / 2;
    const windowHalfY = window.innerHeight / 2;

    document.addEventListener('mousemove', (event) => {
        mouseX = (event.clientX - windowHalfX) * 0.05;
        mouseY = (event.clientY - windowHalfY) * 0.05;
    });

    const clock = new THREE.Clock();
    let isTabActive = true;

    document.addEventListener('visibilitychange', () => {
        isTabActive = !document.hidden;
    });

    function animate() {
        requestAnimationFrame(animate);
        
        if (!isTabActive) return; // Pause rendering if tab is inactive

        const elapsedTime = clock.getElapsedTime();

        // Mouse easing
        targetX = mouseX * 0.1;
        targetY = mouseY * 0.1;
        camera.position.x += (targetX - camera.position.x) * 0.05;
        camera.position.y += (-targetY - camera.position.y) * 0.05;
        camera.lookAt(scene.position);

        // Slow orbit for particles
        particles.rotation.y = elapsedTime * 0.05;
        particles.rotation.x = elapsedTime * 0.02;

        // Animate background sphere slightly
        backgroundSphere.rotation.y = elapsedTime * -0.02;

        renderer.render(scene, camera);
    }

    animate();

    // =========================================================================
    // RESIZE HANDLER
    // =========================================================================
    window.addEventListener('resize', () => {
        camera.aspect = window.innerWidth / window.innerHeight;
        camera.updateProjectionMatrix();
        renderer.setSize(window.innerWidth, window.innerHeight);
    });
});
