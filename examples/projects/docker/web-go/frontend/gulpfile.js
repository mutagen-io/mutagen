// Pull in dependencies.
const { src, dest, watch, parallel } = require('gulp');

// Compute the output path.
const output = process.env.OUTPUT_PATH || "./public";

// Create build rules.
function jqueryJS() {
    return src("node_modules/jquery/dist/jquery.min.*")
        .pipe(dest(output));
}
function popperJS() {
    return src("node_modules/popper.js/dist/umd/popper.min.js*")
        .pipe(dest(output));
}
function bootstrapJS() {
    return src("node_modules/bootstrap/dist/js/bootstrap.min.js*")
        .pipe(dest(output));
}
function bootstrapCSS() {
    return src("node_modules/bootstrap/dist/css/bootstrap.min.css*")
        .pipe(dest(output));
}
function html() {
    return src("index.html")
        .pipe(dest(output));
}
function watchHTML() {
    // We use a watch on the directory rather than the HTML file because gulp's
    // watching can only detect in-place file modifications, not atomic
    // rename-based updates.
    watch(".", html);
}

// Set the default target.
exports.default = parallel(jqueryJS, popperJS, bootstrapJS, bootstrapCSS, html, watchHTML);
