package mockapp

import (
	"io"
	"strings"
)

var LoremIpsumReader io.ReadSeeker

var loremIpsm = `
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aliquam a accumsan
nibh, et ullamcorper augue. Nulla varius risus quis dui mollis, eu ultricies
sem pharetra. Quisque id condimentum quam. Pellentesque eros ante, pretium vel
velit nec, luctus semper lorem. Maecenas convallis viverra tincidunt. Nulla nec
diam vel ex fringilla cursus vitae eget nunc. Nulla interdum ante vitae neque
mattis tincidunt. Cras vel augue et velit molestie fermentum. Phasellus
accumsan turpis imperdiet risus pulvinar iaculis efficitur in est.

Cras ut elementum tortor. Duis varius tellus a luctus tincidunt. Sed elementum
sem ut tempor blandit. Nunc eget tempus diam, ut congue lectus. Maecenas ligula
urna, auctor non dapibus venenatis, vulputate vitae quam. Nullam placerat
bibendum tempus. Etiam cursus elit velit, rutrum gravida dolor tempus eget.
Pellentesque pellentesque mauris metus, in dictum diam euismod non. Ut tempus
vehicula nisi. Sed neque erat, auctor quis elit vitae, congue convallis urna.
Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac
turpis egestas.

Nunc tortor ante, tempor eu congue at, finibus ut risus. Quisque eu diam nec
enim finibus vehicula nec quis odio. Proin a mauris dictum ante elementum
interdum in non lorem. Maecenas quis elit ac quam condimentum dictum. Cras
blandit convallis est eget rutrum. Nulla facilisi. Sed fermentum fringilla ex
vitae molestie.

Mauris nec euismod ligula. Mauris fermentum purus quis justo fermentum blandit.
Pellentesque felis massa, placerat non vulputate a, elementum sit amet nulla.
Praesent lobortis risus lectus, et faucibus tellus euismod id. Nullam eget
turpis varius, facilisis erat aliquet, fermentum ex. Praesent convallis, lacus
dictum auctor semper, libero leo suscipit justo, nec pharetra mi justo in enim.
Nulla sodales scelerisque ligula a dictum. Donec gravida efficitur nulla at
posuere. Nam mattis, neque sed tristique tempor, mauris justo aliquam nibh, non
ullamcorper tortor mauris nec turpis. In hac habitasse platea dictumst.
Maecenas non felis ultrices, cursus tellus ut, cursus est. Mauris eleifend eget
erat eget rhoncus. Phasellus sollicitudin, ante eget sollicitudin volutpat,
metus lorem iaculis nisl, vel semper leo dolor et sapien.

Nullam ornare tempor nulla vel eleifend. Suspendisse tincidunt, tortor vel
tincidunt maximus, arcu orci porta ex, et gravida nibh dolor id purus. Donec
risus nibh, aliquam at ultrices in, facilisis quis ante. Nam ut auctor eros, in
pharetra diam. Proin at tortor tellus. Integer venenatis ut nulla a viverra.
Pellentesque ac nibh ultrices, gravida justo ut, placerat urna. Mauris sagittis
quis massa et blandit. Sed et fringilla risus. Suspendisse malesuada vulputate
dolor.

Donec pellentesque euismod lectus. Etiam nec est velit. Nam diam justo,
lobortis ut accumsan vitae, bibendum ut velit. Ut vehicula est quis enim
pharetra, vitae sagittis nisl efficitur. Praesent at volutpat velit. Sed at
pharetra odio. Sed tincidunt elit in faucibus lobortis. Sed ullamcorper at
purus vel vehicula. Sed a nibh ut sapien luctus blandit. Aenean pellentesque
turpis non sem ullamcorper, ut convallis enim congue. Aliquam volutpat et diam
sed volutpat. Proin luctus dolor tortor, sit amet consectetur enim pulvinar id.
Quisque facilisis tristique dui iaculis gravida. Aliquam at felis mauris.

Duis venenatis odio diam, at interdum ante tempus eget. Fusce malesuada nulla
lorem, eu sollicitudin lectus rutrum a. Donec dignissim at eros laoreet
sollicitudin. Donec malesuada auctor neque, vel viverra lorem malesuada eget.
Fusce enim urna, sagittis ac tristique vitae, iaculis vel elit. Nunc dignissim
lacinia est quis tincidunt. Pellentesque quis felis at quam porttitor pulvinar
non et risus. Integer varius diam eu erat fermentum, quis mattis ligula
posuere. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices
posuere cubilia curae; Curabitur leo orci, suscipit vel porta vitae, facilisis
sit amet eros. Donec lacinia venenatis neque, ac consequat arcu vestibulum
quis. Maecenas luctus nisl sit amet risus facilisis imperdiet. Mauris vel
egestas nisl. Pellentesque consequat blandit leo sit amet mollis. Interdum et
malesuada fames ac ante ipsum primis in faucibus.

Ut tempus scelerisque nisl, at finibus mi maximus commodo. Nullam eleifend ac
erat non pretium. Duis sit amet urna dui. Curabitur ac dui enim. Nunc tempor
lobortis lacus, a auctor odio rutrum eu. Nullam imperdiet tempus orci.
Curabitur luctus diam a molestie pulvinar. Mauris in sapien eget dolor
consectetur tempus. Donec ac nisl quis lectus elementum sollicitudin non vel
quam. Praesent eu magna non sapien tincidunt semper vitae et magna. In metus
libero, maximus rhoncus quam eget, egestas mollis nisl. Donec massa quam,
eleifend ut nulla non, maximus dictum nunc. In hac habitasse platea dictumst.
Maecenas at turpis sit amet turpis convallis bibendum.

Nullam mauris nunc, dictum sed rhoncus ut, varius at felis. Morbi et nibh sed
mauris pretium viverra at nec est. Nam elementum, neque vel tempor tincidunt,
nisl odio efficitur ex, vitae consequat neque turpis ac libero. Pellentesque
rhoncus neque lorem, id tincidunt nisi pellentesque ac. Nulla facilisi. Donec
sodales sodales libero, luctus semper tortor pharetra id. Vivamus ac neque
tincidunt, porttitor lorem id, rhoncus lectus. Vestibulum velit nibh, pretium
placerat ultrices eu, lobortis nec elit.

Maecenas hendrerit sem eros, sed vulputate ante dapibus vitae. Curabitur
vestibulum gravida est eu aliquam. Proin ac feugiat ligula. Nam dolor justo,
efficitur quis fermentum ultricies, luctus ac libero. Morbi non lectus ac risus
rutrum cursus. Pellentesque accumsan hendrerit enim, et commodo dui tempor eu.
Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere
cubilia curae; Nulla facilisi. Integer aliquet rhoncus erat, a facilisis nulla
pellentesque sit amet. Sed nec nisi vitae enim egestas lacinia. Sed at finibus
turpis. Ut eget augue sodales tortor congue bibendum quis vel est. Integer a
pulvinar nibh, sed pretium urna. Nullam molestie eros neque, vel congue tellus
commodo non. Quisque efficitur odio et metus pretium finibus. Mauris aliquam
nunc ut nibh aliquet, quis auctor nulla malesuada.

Fusce tempus hendrerit turpis sit amet placerat. Curabitur gravida tortor
euismod leo pretium, non finibus diam venenatis. Proin non ex eu libero blandit
pulvinar eget vitae lacus. Suspendisse quis tristique orci. Donec a ornare
magna. In vitae faucibus justo. Proin venenatis feugiat enim non venenatis.
Duis non arcu turpis. Vestibulum lacinia augue massa, ac scelerisque leo
tincidunt eu. Nunc pulvinar ante ornare nisi tempor ornare. Ut id nunc quis
urna tincidunt fringilla lacinia bibendum lorem. Cras posuere a nisl auctor
aliquam. Sed et ornare leo. Etiam sagittis tempus est, sed ornare nisi porta
non. Praesent justo mi, venenatis vehicula massa fermentum, elementum dignissim
sapien.

Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere
cubilia curae; Aliquam at vestibulum arcu, eget malesuada neque. Praesent
feugiat, magna at rhoncus elementum, felis orci ornare lorem, suscipit rutrum
ex ex ut massa. In hac habitasse platea dictumst. Morbi euismod nulla non urna
placerat, eu rutrum sapien mollis. Duis feugiat lectus vitae enim consectetur
hendrerit. Ut nec magna justo. Morbi sed dolor at magna dapibus dignissim. Sed
iaculis, felis posuere laoreet vulputate, ex nulla blandit ligula, quis maximus
turpis neque id turpis. Mauris ut cursus ante. Sed ullamcorper quam et
ultricies euismod. Sed id ullamcorper mauris. Duis leo est, cursus in mauris
eget, sodales tincidunt felis. Nam feugiat, nisl nec tincidunt feugiat, metus
enim pellentesque lectus, semper aliquam mi nunc a urna. Quisque sollicitudin
enim a volutpat molestie. Quisque pellentesque justo velit, vitae laoreet nulla
condimentum sed.

Etiam fringilla tortor non erat semper elementum. Sed pretium lobortis nulla.
Nunc est neque, venenatis quis egestas id, tristique ut tortor. Vestibulum et
mauris a nibh commodo dictum. Pellentesque lacinia quam id iaculis convallis.
Praesent consectetur feugiat sapien ac cursus. Mauris condimentum orci diam,
sed imperdiet velit tempus blandit. Aliquam velit aliquam.
`

func init() {
	LoremIpsumReader = strings.NewReader(loremIpsm)
}
