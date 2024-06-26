  // Function to toggle dark mode
  const toggleDarkMode = () => {
    const html = document.querySelector('html');
    const currentTheme = html.getAttribute('data-bs-theme');

    if (currentTheme === 'dark') {
      html.removeAttribute('data-bs-theme');
      darkModeToggle.textContent = 'Dark Mode';
      localStorage.setItem('theme', 'light'); // Store the theme in localStorage
    } else {
      html.setAttribute('data-bs-theme', 'dark');
      darkModeToggle.textContent = 'Light Mode';
      localStorage.setItem('theme', 'dark'); // Store the theme in localStorage
    }
  };

  const darkModeToggle = document.getElementById('darkModeToggle');

  // Check if theme preference is stored in localStorage
  const storedTheme = localStorage.getItem('theme');
  if (storedTheme) {
    const html = document.querySelector('html');
    html.setAttribute('data-bs-theme', storedTheme);
    if (storedTheme === 'dark') {
      darkModeToggle.textContent = 'Light Mode';
    }
  }

  darkModeToggle.addEventListener('click', toggleDarkMode);