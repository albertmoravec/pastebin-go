let gulp = require('gulp');
let sass = require('gulp-sass');
let cleanCSS = require('gulp-clean-css');
let concat = require('gulp-concat');
let uglify = require('gulp-uglify');

gulp.task('default', function() {
    gulp.src('assets/materialize/sass/materialize.scss')
        .pipe(sass())
        .pipe(gulp.dest('assets/css'));

    gulp.src('assets/css/*.css')
        .pipe(cleanCSS())
        .pipe(concat('style.min.css'))
        .pipe(gulp.dest('assets/public/css/'));

    gulp.src([
        "jquery.min.js",
        "velocity.min.js",
        "hammer.min.js",
        "jquery.hammer.js",
        "jquery.easing.1.3.js",
        "global.js",
        "animation.js",
        // "sideNav.js",
        "modal.js",
        "tooltip.js",
        "waves.js",
        "buttons.js",
        "forms2.js",
        "dropdown.js",
        // "autocomplete.js",
        "toasts.js"
    ], {cwd: 'assets/materialize/js/'})
        .pipe(concat('materialize.js'))
        .pipe(gulp.dest('assets/js'));
-
    gulp.src([
        'materialize.js',
        "cookie.js",
        "codemirror.js",
        "javascript.js",
        "simplescrollbar.js",
        "paste.js",
        "matchbrackets.js",
        "simple.js"
    ],{cwd: 'assets/js/'})
        .pipe(uglify())
        .pipe(concat('script.min.js'))
        .pipe(gulp.dest('assets/public/js/'));

});