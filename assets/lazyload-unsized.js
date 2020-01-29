function loadImage (el) {
	var img = new Image(), src = el.getAttribute('data-src');
	img.onload = function() {
		console.log("Image finished Loading");
		if (!! el.parent) {
			el.parent.replaceChild(img, el)
		} else {
			el.src = src;
		}
                el.classList.toggle("g-img");
		finishLoadingImage(el)
	}
	img.onerror = function() {
		failedLoadingImage(el);
	}
	img.src = src;

//        var src = el.getAttribute('data-src');
//        el.onload = function() {
//            el.classList.toggle("g-img");
//            finishLoadingImage(el);
//        }
//        el.src = src;
}

function loadImages() {
        console.log("Loading the next bunch");
        var no_images_load = 45;
        var images = $( ".unloaded" );

        if(images.length > 0) {
            document.getElementById("loading-box").style.display="inline";
        } else {
            finishLoadingImages();
        }
        if(images.length < no_images_load){
            no_images_load = images.length;
        }
	for (i = 0; i < no_images_load ; i++) {
		loading_images += 1;
		loadImage(images[i])
	}
}

function failedLoadingImage (el) {
	console.log("Error loading image");
	//el.innerHTML="";
	// remove unloaded class so we don't keep retrying
	el.classList.remove("unloaded");
	el.classList.add("failed");
	loading_images -= 1;
	if (loading_images < 1) {
		finishLoadingImages();
	}
}

function finishLoadingImage (el) {
	//console.log(el);
	el.classList.remove("unloaded");
	loading_images -= 1;
	if (loading_images < 1) {
		finishLoadingImages();
	}
}

function finishLoadingImages () {
	document.getElementById("loading-box").style.display="none"; // removes loading placeholder
	console.log("Finished Loading Images");
	loading_images_started = false;
}

function checkScroll () {
	var body = document.body, html = document.documentElement;
	var documentHeight = Math.max( body.scrollHeight, body.offsetHeight,html.clientHeight, html.scrollHeight, html.offsetHeight );
	console.log( "window.pageYOffset: " + window.pageYOffset + " documentHeight " + documentHeight + " window.innerHeight: " + window.innerHeight );
	if ( window.pageYOffset + window.innerHeight > (documentHeight - 500) && !loading_images_started ) {
		loading_images_started = true
		loadImages();
	}
}

var loading_images_started = false;
var loading_images = 0;

function init () {
    loadImages();
//    document.getElementById('container').addEventListener('scroll', checkScroll);
    window.onscroll = checkScroll; // only check after the page has loaded
}
window.onload = init;
